package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

// Renderer handles template rendering
type Renderer struct {
	templates *template.Template
}

// NewRenderer creates a new renderer
func NewRenderer(views fs.FS) (*Renderer, error) {
	// Parse all templates
	// Parse all templates manually to preserve paths
	tmpl := template.New("").Funcs(template.FuncMap{
		"isJSONObject": func(v interface{}) bool {
			_, ok := v.(map[string]interface{})
			return ok
		},
		"prettyPrint": func(v interface{}) string {
			if b, err := json.MarshalIndent(v, "", "  "); err == nil {
				return string(b)
			}
			return fmt.Sprint(v)
		},
	})

	err := fs.WalkDir(views, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Read file content
		content, err := fs.ReadFile(views, path)
		if err != nil {
			return err
		}

		// Parse template with path as name
		_, err = tmpl.New(path).Parse(string(content))
		return err
	})
	if err != nil {
		return nil, err
	}



	return &Renderer{
		templates: tmpl,
	}, nil
}

// Render renders a template
func (r *Renderer) Render(w http.ResponseWriter, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Ensure name ends with .html
	if !strings.HasSuffix(name, ".html") {
		name = name + ".html"
	}
	
	// Template names in ParseFS are relative to the root of FS
	// e.g. "index.html", "docs/api/overview.html"
	
	return r.templates.ExecuteTemplate(w, name, data)
}
