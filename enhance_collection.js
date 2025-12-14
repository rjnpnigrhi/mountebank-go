
const fs = require('fs');
const http = require('http');
const https = require('https');
const url = require('url');

const collectionPath = "mountebank_postman_collection.json";
const baseUrl = "http://localhost:2525";
const imposterPort = 4545;

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// Function to resolve variables in URL/Body
function resolveVariables(str) {
    if (typeof str !== 'string') return str;
    return str.replace(/{{baseUrl}}/g, baseUrl)
        .replace(/{{imposterPort}}/g, imposterPort);
}

// Function to execute request
function executeRequest(item) {
    return new Promise((resolve, reject) => {
        const reqDef = item.request;
        const method = reqDef.method;

        // Construct URL
        let reqUrlStr = "";
        if (reqDef.url.raw) {
            reqUrlStr = resolveVariables(reqDef.url.raw);
        } else if (typeof reqDef.url === 'string') {
            reqUrlStr = resolveVariables(reqDef.url);
        } else {
            // Url object
            const prot = reqDef.url.protocol || 'http';
            const host = reqDef.url.host ? reqDef.url.host.join('.') : 'localhost';
            const path = reqDef.url.path ? reqDef.url.path.join('/') : '';
            const query = reqDef.url.query ? '?' + reqDef.url.query.map(q => `${q.key}=${q.value}`).join('&') : '';
            reqUrlStr = resolveVariables(`${prot}://${host}/${path}${query}`);
        }

        // Parse URL
        // If relative URL (which shouldn't happen with raw usually, but postman allows it?)
        // Assuming absolute or {{baseUrl}} which we resolved.
        if (!reqUrlStr.startsWith('http')) {
            reqUrlStr = baseUrl + (reqUrlStr.startsWith('/') ? '' : '/') + reqUrlStr;
        }

        const parsedUrl = url.parse(reqUrlStr);
        const options = {
            hostname: parsedUrl.hostname,
            port: parsedUrl.port,
            path: parsedUrl.path,
            method: method,
            headers: {
                'Content-Type': 'application/json' // Default
            }
        };

        if (reqDef.header) {
            reqDef.header.forEach(h => {
                options.headers[h.key] = resolveVariables(h.value);
            });
        }

        let body = "";
        if (reqDef.body && reqDef.body.mode === 'raw' && reqDef.body.raw) {
            body = resolveVariables(reqDef.body.raw);
            options.headers['Content-Length'] = Buffer.byteLength(body);
        }

        // console.log(`Executing ${method} ${reqUrlStr}`);

        const lib = parsedUrl.protocol === 'https:' ? https : http;
        const req = lib.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => data += chunk);
            res.on('end', () => {
                // Parse JSON if possible
                let parsedBody = data;
                try {
                    parsedBody = JSON.parse(data);
                } catch (e) {
                    // Not JSON
                }

                resolve({
                    statusCode: res.statusCode,
                    headers: res.headers,
                    body: parsedBody,
                    rawBody: data
                });
            });
        });

        req.on('error', (e) => {
            console.error(`Error requesting ${reqUrlStr}: ${e.message}`);
            // Resolve with null to skip enhancement but continue
            resolve(null);
        });

        if (body) {
            req.write(body);
        }
        req.end();
    });
}

// Function to generate recursive key check script
function generateKeyCheckScript(body) {
    if (typeof body !== 'object' || body === null) return "";

    const jsonStr = JSON.stringify(body);

    return `
    pm.test("Strict Key Check (Full Depth)", function () {
        var jsonData = pm.response.json();
        var expected = ${jsonStr};
        
        function checkKeys(exp, act, path) {
            if (exp === null || act === null) return;
            if (typeof exp !== 'object') return;
            
            // If array, iterate items if needed? 
            // The requirement is "presense of each fields". 
            // If array, we probably check if actual is array and optionally check matching items structure.
            // For simplicity and "strictness" matching specific response:
            if (Array.isArray(exp)) {
                pm.expect(act).to.be.an('array', path + " should be array");
                // We might check length? 
                // Let's check keys of the FIRST item if it exists, assuming homogeneous array? 
                // Or just check that act has at least same length?
                // Strict parity implies identical structure. 
                // If the response is a LIST of things, checking strict properties of each might be too brittle if list order changes?
                // But user wants "parity".
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

// Function to generate header presence check script
function generateHeaderCheckScript(headers) {
    let script = `
    pm.test("Strict Header Presence", function () {
        var expectedHeaders = ${JSON.stringify(Object.keys(headers))};
        expectedHeaders.forEach(function(header) {
            // Skip dynamic headers that might be transient (though user asked for ALL headers)
            // But 'Date' or 'Connection' are usually fine to check PRESENCE.
            pm.response.to.have.header(header);
        });
    });`;
    return script;
}

// Recursive function to process collection items
async function processItems(items) {
    for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.item) {
            // Folder
            console.log(`Folder: ${item.name}`);
            await processItems(item.item);
        } else if (item.request) {
            // Request
            console.log(`Request: ${item.name}`);

            // Wait a bit to avoid overwhelming? (optional)
            await sleep(50);

            const response = await executeRequest(item);

            if (response) {
                // Generate Scripts
                let testScripts = [];

                // Existing scripts? We append or replace?
                // User said "enhance". Let's append if possible, but we want to ensure we check presence.
                // We'll create a new "event" array or append to "test" event.

                if (!item.event) item.event = [];
                let testEvent = item.event.find(e => e.listen === 'test');
                if (!testEvent) {
                    testEvent = { listen: 'test', script: { type: 'text/javascript', exec: [] } };
                    item.event.push(testEvent);
                }

                // Parse existing exec (array of strings)
                let execLines = testEvent.script.exec || [];

                // Add Status Check if not exists (usually exists)
                // Add Strict Header Check
                const headerCheck = generateHeaderCheckScript(response.headers);

                // Add Strict Body Check (if JSON)
                let bodyCheck = "";
                if (typeof response.body === 'object') {
                    bodyCheck = generateKeyCheckScript(response.body);
                }

                // We add these new tests to the script
                // To avoid duplication if we run multiple times, maybe add a marker?
                // For now, assume single run.

                // Convert script strings to array of lines for Postman
                const headerLines = headerCheck.split('\n');
                const bodyLines = bodyCheck.split('\n');

                execLines.push(...headerLines);
                execLines.push(...bodyLines);

                testEvent.script.exec = execLines;

                console.log(`  Enhanced ${item.name} with assertions.`);
            } else {
                console.log(`  Failed to execute ${item.name}, skipping enhancement.`);
            }
        }
    }
}

async function main() {
    try {
        const content = fs.readFileSync(collectionPath, 'utf8');
        const collection = JSON.parse(content);

        await processItems(collection.item);

        fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 4));
        console.log("Collection enhanced successfully.");
    } catch (e) {
        console.error(e);
    }
}

main();
