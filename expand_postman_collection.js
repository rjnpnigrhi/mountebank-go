
const fs = require('fs');

const collectionPath = "mountebank_postman_collection.json";
const imposterPort = 4545;
const baseUrl = "{{baseUrl}}";

function createSetupItem() {
    return {
        "name": `Setup Imposter ${imposterPort}`,
        "request": {
            "method": "PUT",
            "header": [{ "key": "Content-Type", "value": "application/json" }],
            "body": {
                "mode": "raw",
                "raw": JSON.stringify({
                    imposters: [{ port: imposterPort, protocol: "http", stubs: [{ responses: [{ is: { statusCode: 200 } }] }] }]
                }, null, 2)
            },
            "url": {
                "raw": `${baseUrl}/imposters`,
                "host": ["{{baseUrl}}"],
                "path": ["imposters"]
            }
        },
        "event": [{
            "listen": "test",
            "script": { "exec": ["pm.test(\"Setup: Status 200\", function(){ pm.response.to.have.status(200); });"], "type": "text/javascript" }
        }]
    };
}

function createRequest(name, method, urlPath, body = null, description = "") {
    const item = {
        "name": name,
        "request": {
            "method": method,
            "header": [{ "key": "Content-Type", "value": "application/json" }],
            "url": {
                "raw": `${baseUrl}${urlPath}`,
                "host": ["{{baseUrl}}"],
                "path": urlPath.split('/').filter(p => p.length > 0)
            }
        },
        "event": [{
            "listen": "test",
            "script": { "exec": ["pm.test(\"Status code is 200\", function(){ pm.response.to.have.status(200); });"], "type": "text/javascript" }
        }]
    };

    if (body) {
        item.request.body = {
            "mode": "raw",
            "raw": JSON.stringify(body, null, 2)
        };
    }
    return item;
}

const missingApis = [
    {
        name: "GET Config",
        method: "GET",
        url: "/config"
    },
    {
        name: "GET Logs",
        method: "GET",
        url: "/logs"
    },
    {
        name: "GET Logs (startIndex)",
        method: "GET",
        url: "/logs?startIndex=0"
    },
    {
        name: "GET Logs (endIndex)",
        method: "GET",
        url: "/logs?endIndex=5"
    },
    {
        name: "PUT Stubs (Overwrite All)",
        method: "PUT",
        url: `/imposters/${imposterPort}/stubs`,
        body: { stubs: [{ responses: [{ is: { statusCode: 201 } }] }] }
    },
    {
        name: "POST Stub (Add New)",
        method: "POST",
        url: `/imposters/${imposterPort}/stubs`,
        body: { stub: { responses: [{ is: { statusCode: 404 } }] } }
    },
    {
        name: "PUT Stub at Index 0",
        method: "PUT",
        url: `/imposters/${imposterPort}/stubs/0`,
        body: { responses: [{ is: { statusCode: 400 } }] }
    },
    {
        name: "DELETE Stub at Index 0",
        method: "DELETE",
        url: `/imposters/${imposterPort}/stubs/0`
    },
    {
        name: "DELETE Saved Proxy Responses",
        method: "DELETE",
        url: `/imposters/${imposterPort}/savedProxyResponses`
    },
    {
        name: "DELETE Saved Requests",
        method: "DELETE",
        url: `/imposters/${imposterPort}/savedRequests`
    },
    // Protocol implementation APIs (Mountebank uses these internally usually, but exposed)
    {
        name: "POST _requests (Simulate)",
        method: "POST",
        url: `/imposters/${imposterPort}/_requests`,
        body: { request: { method: "GET", path: "/", query: {}, headers: {} } }
    }
];

function main() {
    try {
        const content = fs.readFileSync(collectionPath, 'utf8');
        const collection = JSON.parse(content);

        const newFolder = {
            "name": "Missing APIs Coverage",
            "item": []
        };

        missingApis.forEach(api => {
            // Add setup for context-dependent APIs
            if (api.url.includes(imposterPort.toString())) {
                newFolder.item.push(createSetupItem());
            }

            // Handle query strings in url object construction
            const req = createRequest(api.name, api.method, api.url.split('?')[0]);
            if (api.body) req.request.body = { mode: "raw", raw: JSON.stringify(api.body, null, 2) };

            // Query params
            if (api.url.includes('?')) {
                const queryStr = api.url.split('?')[1];
                const queries = queryStr.split('&').map(q => {
                    const parts = q.split('=');
                    return { key: parts[0], value: parts[1] };
                });
                req.request.url.query = queries;
                req.request.url.raw = `${baseUrl}${api.url}`;
            }

            // Adjust assertions for POST/201 where appropriate
            if (api.method === "POST" && api.name !== "POST _requests (Simulate)") {
                // Note: POST /stubs returns imposter JSON, usually 200 OK ? Controller says: response.send(json) -> default 200
                // POST /imposters returns 201.
                // Let's stick to 200 check unless we know better.
                // POST /stubs returns updated imposter json.
            }

            newFolder.item.push(req);
        });

        collection.item.push(newFolder);

        fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 4));
        console.log("Added Missing APIs to collection.");

    } catch (e) {
        console.error(e);
    }
}

main();
