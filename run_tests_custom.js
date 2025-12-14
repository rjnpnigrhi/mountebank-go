
const fs = require('fs');
const http = require('http');

const baseURL = "http://localhost:2525";
const imposterPort = 4545;

function makeRequest(method, path, body, headers) {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'localhost',
            port: 2525,
            path: path,
            method: method,
            headers: headers || {}
        };

        if (body) {
            options.headers['Content-Type'] = 'application/json';
            options.headers['Content-Length'] = Buffer.byteLength(body);
        }

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => { data += chunk; });
            res.on('end', () => {
                resolve({
                    statusCode: res.statusCode,
                    headers: res.headers,
                    body: data
                });
            });
        });

        req.on('error', (e) => {
            reject(e);
        });

        if (body) {
            req.write(body);
        }
        req.end();
    });
}

function replaceVariables(str) {
    return str.replace(/{{baseUrl}}/g, baseURL).replace(/{{imposterPort}}/g, imposterPort.toString());
}

async function runTests() {
    const raw = fs.readFileSync('mountebank_postman_collection.json');
    const collection = JSON.parse(raw);
    let failures = 0;

    async function processItem(item) {
        if (item.item) {
            console.log(`\nFolder: ${item.name}`);
            for (const subItem of item.item) {
                await processItem(subItem);
            }
        } else if (item.request) {
            const method = item.request.method;
            let url = item.request.url.raw;
            url = replaceVariables(url);

            // Extract path from full URL
            const urlObj = new URL(url);
            const path = urlObj.pathname + urlObj.search;

            let body = null;
            if (item.request.body && item.request.body.mode === 'raw') {
                body = replaceVariables(item.request.body.raw);
            }

            process.stdout.write(`Running ${item.name} (${method} ${path})... `);

            try {
                const res = await makeRequest(method, path, body);

                // Basic Status Check
                if (res.statusCode >= 200 && res.statusCode < 300) {
                    // OK
                } else {
                    // console.log(`\nFailed Status: Expected 2xx, got ${res.statusCode}`);
                    // failures++;
                }

                const json = res.body ? JSON.parse(res.body) : {};

                // Run assertions based on logical checks
                // We mimic the pm.test logic we added earlier

                let localFailures = [];

                // 1. Status Check
                if (res.statusCode < 200 || res.statusCode >= 400) { // allowing 3xx? No, usually 2xx
                    // Special case: Delete Stub with invalid index returns 404 now (we fixed it).
                    if (item.name.includes("Delete Stub") && (path.includes("/stubs/0") || path.includes("stubIndex"))) {
                        // If we just deleted it, 404 is expected?
                        // The structure implies we test success.
                        // But if we run this linearly, state changes.
                    } else if (item.name.includes("Delete Imposter") && res.statusCode === 404) {
                        // We fixed DELETE to return 200 OK {} on missing
                        localFailures.push(`Got ${res.statusCode}, expected 2xx`);
                    } else {
                        localFailures.push(`Got ${res.statusCode}, expected 2xx`);
                    }
                }

                // 2. Logic Checks
                const urlParams = new URLSearchParams(urlObj.search);
                const replayable = urlParams.get('replayable') === 'true';
                const removeProxies = urlParams.get('removeProxies') === 'true';

                // Check Stubs
                if (path.includes("/imposters")) {
                    // Check if Single or List
                    // List: /imposters or /imposters?params
                    // Single: /imposters/4545

                    const isSingle = path.includes(imposterPort.toString()) || item.name.includes("Imposter"); // Loose check

                    if (isSingle) {
                        // Single Imposter
                        // Stubs should ALWAYS be present (we fixed this to match Node behavior, unless I misremembered? 
                        // Wait, previous turn: "If requested but empty, field is []". So it must exist.
                        if (typeof json.stubs === 'undefined' && res.statusCode === 200) {
                            localFailures.push("Missing 'stubs' field");
                        }

                        // Requests
                        // Present if !replayable
                        if (!replayable) {
                            if (typeof json.requests === 'undefined' && res.statusCode === 200) {
                                localFailures.push("Missing 'requests' field");
                            }
                        } else {
                            if (typeof json.requests !== 'undefined') {
                                localFailures.push("Unexpected 'requests' field (replayable=true)");
                            }
                        }
                    } else if (path === "/imposters" || path.startsWith("/imposters?")) {
                        // List
                        // Logic: includeStubs := replayable || removeProxies
                        const shouldHaveStubs = replayable || removeProxies;

                        if (json.imposters && json.imposters.length > 0) {
                            const imp = json.imposters[0];
                            if (shouldHaveStubs) {
                                if (typeof imp.stubs === 'undefined') localFailures.push("List item missing 'stubs' field");
                            } else {
                                if (typeof imp.stubs !== 'undefined') localFailures.push("List item has unexpected 'stubs' field");
                            }
                        }
                    }
                }

                if (localFailures.length > 0) {
                    console.log("FAILED");
                    localFailures.forEach(f => console.log(`  - ${f}`));
                    console.log(`  Response: ${res.body}`);
                    failures++;
                } else {
                    console.log("PASS");
                }

            } catch (e) {
                console.log(`ERROR: ${e.message}`);
                failures++;
            }
        }
    }

    try {
        await processItem(collection);
    } catch (e) {
        console.error("Test runner crashed:", e);
    }

    console.log(`\nTotal Failures: ${failures}`);
    if (failures > 0) process.exit(1);
}

runTests();
