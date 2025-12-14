
const fs = require('fs');
const collectionPath = "mountebank_postman_collection.json";
const reportPath = "report.json";

// Function to generate recursive key check script
function generateKeyCheckScript(body) {
    if (typeof body !== 'object' || body === null) return "";

    // We want strict presence.
    // If array, we check it is array.
    // If object, we check keys.
    const jsonStr = JSON.stringify(body);

    // Check if body is Array?
    if (Array.isArray(body)) {
        return `
    pm.test("Strict Body Check (Array)", function () {
        var jsonData = pm.response.json();
        pm.expect(jsonData).to.be.an('array');
        // We can check length or item structure if we want deep strictness
        // The user said "presense of each fields full depth".
        // This usually implies object structure.
        // For array, it implies verifying items match expected schema?
        // Since we are snapshotting, strict equality of structure is good.
        // But let's stick to "presence of fields" interpretation.
        // If array, we check if items have fields?
        var expected = ${jsonStr};
        if (expected.length > 0) {
             // Check first item structure
             // This assumes homogeneous array.
             var expItem = expected[0];
             var actItem = jsonData[0];
             if (actItem) {
                 checkKeys(expItem, actItem, "root[0]");
             } else {
                 // Actual array empty but expected not?
                 pm.expect(jsonData.length).to.be.at.least(1, "Expected array to have items");
             }
        }
        
        function checkKeys(exp, act, path) {
            if (exp === null || act === null) return;
            if (typeof exp !== 'object') return;
            
            for (var key in exp) {
                if (exp.hasOwnProperty(key)) {
                    pm.expect(act).to.have.property(key, undefined, path + "." + key + " missing");
                    if (typeof exp[key] === 'object' && exp[key] !== null) {
                         checkKeys(exp[key], act[key], path + "." + key);
                    }
                }
            }
        }
    });`;
    }

    // Object
    return `
    pm.test("Strict Key Check (Full Depth)", function () {
        var jsonData = pm.response.json();
        var expected = ${jsonStr};
        
        function checkKeys(exp, act, path) {
            if (exp === null || act === null) return;
            if (typeof exp !== 'object') return;
            
            if (Array.isArray(exp)) {
                pm.expect(act).to.be.an('array', path + " should be array");
                if (exp.length > 0 && act.length > 0) {
                     checkKeys(exp[0], act[0], path + "[0]");
                }
                return;
            }
            
            for (var key in exp) {
                if (exp.hasOwnProperty(key)) {
                    pm.expect(act).to.have.property(key, undefined, path + "." + key + " missing");
                    if (typeof exp[key] === 'object' && exp[key] !== null) {
                         checkKeys(exp[key], act[key], path + "." + key);
                    }
                }
            }
        }
        checkKeys(expected, jsonData, "root");
    });`;
}

function generateHeaderCheckScript(headers) {
    // Headers in report might be list of objects {key, value} or object {key: value}
    // Newman report usually has array of headers.
    let headerNames = [];
    if (Array.isArray(headers)) {
        headerNames = headers.map(h => h.key);
    } else if (typeof headers === 'object') {
        headerNames = Object.keys(headers);
    }

    return `
    pm.test("Strict Header Presence", function () {
        var expectedHeaders = ${JSON.stringify(headerNames)};
        expectedHeaders.forEach(function(header) {
            pm.response.to.have.header(header);
        });
    });`;
}


function main() {
    try {
        const reportContent = fs.readFileSync(reportPath, 'utf8');
        const report = JSON.parse(reportContent);

        const collectionContent = fs.readFileSync(collectionPath, 'utf8');
        const collection = JSON.parse(collectionContent);

        // Map executions to collection items
        // We assume executions are in order of collection traversal (mostly)
        // But we need to match by ID or Name.
        // Collection items have IDs? Not always in exported JSON.
        // Report executions refer to `item` object with `id` and `name`.
        // We need to walk the collection and match executions.

        // Let's index executions by name? Names are not unique (Setups).
        // Let's just traverse report executions and find corresponding item in collection?
        // But collection is nested. Report is flat list of executions.

        // Strategy: Flatten collection to list of Request Items (references).
        // Iterate Report Executions. Match to flatten list sequentially?

        let flatItems = [];
        function flatten(items) {
            items.forEach(item => {
                if (item.item) flatten(item.item);
                if (item.request) flatItems.push(item);
            });
        }
        flatten(collection.item);

        // Report executions
        const executions = report.run.executions;

        // They should match 1-to-1 if newman ran everything.
        if (executions.length !== flatItems.length) {
            console.warn(`Warning: Execution count ${executions.length} != Collection item count ${flatItems.length}. Matching by order might align, but verify.`);
        }

        let matchCount = 0;
        executions.forEach((exec, index) => {
            if (index >= flatItems.length) return;

            // Verify name match
            const item = flatItems[index];
            if (item.name !== exec.item.name) {
                console.warn(`Mismatch at index ${index}: Exec ${exec.item.name} != Item ${item.name}`);
                // Try to find correct item in flatItems?
                // Mountebank collection involves folders. newman order is DFS.
                // flatten order is DFS.
                // Should match.
            }

            // Extract Response
            if (exec.response) {
                // Response body
                let body = null;
                if (exec.response.stream) {
                    const buffer = Buffer.from(exec.response.stream.data);
                    const bodyStr = buffer.toString('utf8');
                    try {
                        body = JSON.parse(bodyStr);
                    } catch (e) {
                        // Not JSON
                    }
                }

                // Generate Scripts
                if (!item.event) item.event = [];
                let testEvent = item.event.find(e => e.listen === 'test');
                if (!testEvent) {
                    testEvent = { listen: 'test', script: { type: 'text/javascript', exec: [] } };
                    item.event.push(testEvent);
                }

                // CLEANUP LOGIC
                let execLines = testEvent.script.exec || [];
                let fullScript = execLines.join('\n');

                // Regex to remove Strict Header block (Orphan or Full)
                // Removes: var expectedHeaders = [...]; ... });
                // Pattern matches non-greedy until end of forEach loop and closing brace?
                // The bad block: var expectedHeaders = [...]; ... });
                fullScript = fullScript.replace(/var expectedHeaders = \[[\s\S]*?\}\);\s*\}\);?/g, "");

                // Regex to remove Strict Key Check block (Orphan or Full)
                // Removes: pm.test("Strict Key Check... });
                fullScript = fullScript.replace(/pm\.test\("Strict [\s\S]*?\}\);/g, "");

                // Remove Orphan expected var blocks: var expected = {...}; ... checkKeys(..., "root"); });
                // Note: The orphan block might start with var jsonData... or var expected...
                // We target the recursive function definition mainly?
                // function checkKeys(exp, act, path) { ... } checkKeys(expected, jsonData, "root"); });
                fullScript = fullScript.replace(/var expected = \{[\s\S]*?checkKeys\(expected, jsonData, "root"\);\s*\}\);?/g, "");

                // Also remove the "Strict Header Presence" full block if usage of pm.test was correct
                // Covered by Strict pm.test regex above.

                // Remove "Strict Body Check (Array)"
                // Covered by Strict pm.test regex.

                // Cleanup extra newlines?
                fullScript = fullScript.replace(/\n\s*\n\s*\n/g, "\n\n");

                // Re-add new scripts
                let newScriptPart = "";

                if (exec.response.header) {
                    newScriptPart += "\n" + generateHeaderCheckScript(exec.response.header);
                }
                if (body) {
                    newScriptPart += "\n" + generateKeyCheckScript(body);
                }

                testEvent.script.exec = (fullScript + newScriptPart).split('\n');
                matchCount++;
            }
        });

        console.log(`Updated ${matchCount} items.`);

        fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 4));
        console.log("Collection updated from report.");

    } catch (e) {
        console.error(e);
    }
}

main();
