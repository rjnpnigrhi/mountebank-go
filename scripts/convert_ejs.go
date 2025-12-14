package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	srcDir := "../src/views"
	destDir := "internal/web/views"

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".ejs" {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, strings.Replace(relPath, ".ejs", ".html", 1))
		destDir := filepath.Dir(destPath)

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		converted := convertEJS(string(content))

		if err := ioutil.WriteFile(destPath, []byte(converted), 0644); err != nil {
			return err
		}

		fmt.Printf("Converted %s to %s\n", path, destPath)
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func convertEJS(content string) string {
	// Manual fix for config.ejs
	// Replace the complex if condition that regex fails to handle correctly
	// Do this BEFORE regexes to avoid interference
	content = strings.ReplaceAll(content, 
		`<% if (isJSONObject(options[key])) { -%>`, 
		`{{ if isJSONObject $value }}`)
	content = strings.ReplaceAll(content,
		`<% if (isJSONObject(options[key])) { %>`,
		`{{ if isJSONObject $value }}`)
	content = strings.ReplaceAll(content,
		`<%= key %>`,
		`{{ $key }}`)
	content = strings.ReplaceAll(content,
		`<%= prettyPrint(options[key]) %>`,
		`{{ prettyPrint $value }}`)

	// Fix for process.env.MB_PERSISTENT in overview.ejs
	// Do this BEFORE regexes to avoid interference
	content = strings.ReplaceAll(content, `<% if (process.env.MB_PERSISTENT === 'true') { %>`, `{{ if false }}`)

	// Manual fix for feed.ejs
	if strings.Contains(content, "<feed xmlns=") {
		content = strings.ReplaceAll(content, 
			`<% if (hasNextPage) { %>`, 
			`{{ if .hasNextPage }}`)
		
		// Stricter replacement
		content = strings.ReplaceAll(content, 
			`<updated><%= releases[0].date %>T00:00:00Z</updated>`, 
			`<updated>{{ (index .releases 0).date }}T00:00:00Z</updated>`)
			
		// Fix loop variables
		content = strings.ReplaceAll(content, `<%= release.version %>`, `{{ $release.version }}`)
		content = strings.ReplaceAll(content, `<%= release.date %>`, `{{ $release.date }}`)
		content = strings.ReplaceAll(content, `<%- release.view %>`, `{{ $release.view }}`)
	}

	// Handle JSON.stringify
	// <%= JSON.stringify(obj, null, 2) %> -> {{ prettyPrint .obj }}
	// Do this BEFORE reVar
	reJson := regexp.MustCompile(`<%= JSON\.stringify\(([^,]+),\s*null,\s*2\)\s*%>`)
	content = reJson.ReplaceAllStringFunc(content, func(match string) string {
		sub := reJson.FindStringSubmatch(match)
		v := strings.TrimSpace(sub[1])
		return fmt.Sprintf(`{{ prettyPrint $%s }}`, v) // Assuming variable is $v from range or .v
	})
	
	// Fix for imposter.ejs specifically if regex misses (e.g. spaces)
	if strings.Contains(content, "imposter.requests.forEach") {
		content = strings.ReplaceAll(content, `<%= JSON.stringify(request, null, 2) %>`, `{{ prettyPrint $request }}`)
		content = strings.ReplaceAll(content, `<%= JSON.stringify(stub, null, 2) %>`, `{{ prettyPrint $stub }}`)
	}

	// Fix for _imposter.ejs
	content = strings.ReplaceAll(content, 
		`<%= imposter.name || imposter.protocol + ':' + imposter.port %>`, 
		`{{ if .imposter.name }}{{ .imposter.name }}{{ else }}{{ .imposter.protocol }}:{{ .imposter.port }}{{ end }}`)

	// Replace includes
	// <%- include('_header') -%> -> {{ template "_header.html" . }}
	// <%- include('../../_header') -%> -> {{ template "_header.html" . }} (flattened? or relative?)
	// Go templates are usually flat namespace if parsed together.
	// Let's assume we register them by name.
	// But wait, if we have subdirectories, names might be "docs/api/overview.html".
	// Mountebank includes seem to be relative.
	// Regex for include
	reInclude := regexp.MustCompile(`<%- include\('([^']+)'\) -%>`)
	content = reInclude.ReplaceAllStringFunc(content, func(match string) string {
		sub := reInclude.FindStringSubmatch(match)
		name := sub[1]
		// Normalize name: remove ../, remove leading /, add .html
		name = filepath.Base(name) + ".html"
		return fmt.Sprintf(`{{ template "%s" . }}`, name)
	})

	// Replace variable interpolation
	// <%= variable %> -> {{ .variable }}
	reVar := regexp.MustCompile(`<%= ([^%]+) %>`)
	content = reVar.ReplaceAllStringFunc(content, func(match string) string {
		sub := reVar.FindStringSubmatch(match)
		v := strings.TrimSpace(sub[1])
		// Handle simple property access
		v = strings.ReplaceAll(v, "()", "")
		return fmt.Sprintf(`{{ .%s }}`, v)
	})

	// Replace if
	// <% if (cond) { %> -> {{ if cond }}
	// This is hard because cond is JS.
	// Example: <% if (notices.length > 0) { %>
	// Go: {{ if gt (len .notices) 0 }}
	// We can't easily translate generic JS.
	// But we can try simple ones.
	
	// Handle Object.keys(map).forEach
	// <% Object.keys(options).forEach(key =>{ %> -> {{ range $key, $value := .options }}
	reObjKeys := regexp.MustCompile(`<% Object\.keys\(([^)]+)\)\.forEach\((\w+)\s*=>\s*\{ %>`)
	content = reObjKeys.ReplaceAllStringFunc(content, func(match string) string {
		sub := reObjKeys.FindStringSubmatch(match)
		mapName := strings.TrimSpace(sub[1])
		keyName := strings.TrimSpace(sub[2])
		// We use $value as well, assuming we want to access it.
		// But the original code uses map[key].
		// Go range over map gives key and value.
		return fmt.Sprintf(`{{ range $%s, $value := .%s }}`, keyName, mapName)
	})

	// Handle forEach with arrow function
	// <% list.forEach(item => { %> -> {{ range $item := .list }}
	reForEachArrow := regexp.MustCompile(`<% (.+)\.forEach\s*\(([^)]+)\s*=>\s*\{ -?%>`)
	content = reForEachArrow.ReplaceAllStringFunc(content, func(match string) string {
		sub := reForEachArrow.FindStringSubmatch(match)
		list := strings.TrimSpace(sub[1])
		item := strings.TrimSpace(sub[2])
		return fmt.Sprintf(`{{ range $%s := .%s }}`, item, list)
	})

	// Handle forEach with function
	// <% list.forEach(function (item) { %> -> {{ range $item := .list }}
	reForEach := regexp.MustCompile(`<% (.+)\.forEach\s*\(function \(([^)]+)\) \{ -?%>`)
	content = reForEach.ReplaceAllStringFunc(content, func(match string) string {
		sub := reForEach.FindStringSubmatch(match)
		list := strings.TrimSpace(sub[1])
		item := strings.TrimSpace(sub[2])
		return fmt.Sprintf(`{{ range $%s := .%s }}`, item, list)
	})

	// Handle closing braces
	// <% }); %> -> {{ end }}
	// <% } %> -> {{ end }}
	// <% }); -%> -> {{ end }}
	reEnd := regexp.MustCompile(`<% }\);? -?%>|<% } -?%>`)
	content = reEnd.ReplaceAllString(content, `{{ end }}`)

	// Handle simple if
	// <% if (x) { %>
	// Use regex that handles one level of nested parens: if (foo(bar)) {
	reIf := regexp.MustCompile(`<% if \(((?:[^()]+|\([^()]*\))+)\) \{ -?%>`)
	content = reIf.ReplaceAllStringFunc(content, func(match string) string {
		sub := reIf.FindStringSubmatch(match)
		cond := strings.TrimSpace(sub[1])
		
		// Hack for config.ejs
		cond = strings.ReplaceAll(cond, "options[key]", "$value")
		
		// Remove outer parens if they exist and match
		// But since we captured inside if (...), we just need to clean up function calls.
		// isJSONObject($value) -> isJSONObject $value
		
		// Simple heuristic: replace ( with space and remove )
		cond = strings.ReplaceAll(cond, "(", " ")
		cond = strings.ReplaceAll(cond, ")", "")
		
		return fmt.Sprintf(`{{ if %s }}`, cond)
	})
	
	// Handle function calls in print
	// <%= prettyPrint(options[key]) %>
	rePrint := regexp.MustCompile(`<%= ([^%]+) %>`)
	content = rePrint.ReplaceAllStringFunc(content, func(match string) string {
		sub := rePrint.FindStringSubmatch(match)
		expr := strings.TrimSpace(sub[1])
		
		// prettyPrint(options[key]) -> prettyPrint $value
		if strings.Contains(expr, "prettyPrint(options[key])") {
			return `{{ prettyPrint $value }}`
		}
		
		// Handle simple property access
		return fmt.Sprintf(`{{ .%s }}`, expr)
	})
	
	// Remove top level JS blocks (variables and functions)
	// <% ... %>
	reBlock := regexp.MustCompile(`(?s)<%[^=].*?%>`)
	content = reBlock.ReplaceAllString(content, "")

	if strings.Contains(content, "config.html") || strings.Contains(content, "Config") {
		// This check is weak because we don't have filename here easily (it's passed to main but not convertEJS)
		// But we can check for unique content of config.ejs
		if strings.Contains(content, "Process Information") {
			fmt.Println("DEBUG: converted config.html:")
			fmt.Println(content)
		}
	}

	// Fix for notices variable in if condition
	content = strings.ReplaceAll(content, `{{ if notices }}`, `{{ if .notices }}`)
	content = strings.ReplaceAll(content, `{{ if notices.length > 0 }}`, `{{ if gt (len .notices) 0 }}`)

	return content
}
