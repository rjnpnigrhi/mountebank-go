
const fs = require('fs');

// Helper to create a Setup Imposter Item
function createSetupItem(port) {
    return {
        "name": `Setup Imposter ${port}`,
        "request": {
            "method": "POST",
            "header": [
                {
                    "key": "Content-Type",
                    "value": "application/json"
                },
                {
                    "key": "Accept",
                    "value": "application/json"
                }
            ],
            "body": {
                "mode": "raw",
                "raw": JSON.stringify({
                    port: parseInt(port),
                    protocol: "http"
                }, null, 2)
            },
            "url": {
                "raw": "{{baseUrl}}/imposters",
                "host": ["{{baseUrl}}"],
                "path": ["imposters"]
            }
        },
        "event": [
            {
                "listen": "test",
                "script": {
                    "exec": [
                        "pm.test(\"Setup: Status code is 201\", function () {",
                        "    pm.response.to.have.status(201);",
                        "});"
                    ],
                    "type": "text/javascript"
                }
            }
        ]
    };
}

function resultHasTest(event) {
    // Basic check if it has a test script
    return event && event.length > 0 && event[0].listen === 'test';
}

function processFolder(folder) {
    const newItems = [];
    const imposterPort = 4545; // Default from collection variables usually

    // 1. Initial Setup for the whole folder (for GET tests)
    // We'll treat this folder as a "Suite"

    // We can group consecutive READ operations and put one Setup before them.
    // We MUST put a Setup before EVERY destructive (DELETE) operation.

    // Filter existing setup items if we run this script multiple times? 
    // Ideally we assume clean slate from the previous generated state or we clean up names like "Setup Imposter..."

    // Let's iterate original items
    const startItems = folder.item.filter(i => !i.name.startsWith("Setup Imposter"));

    // Setup for the initial block
    newItems.push(createSetupItem(imposterPort));

    startItems.forEach(item => {
        const method = item.request ? item.request.method : "";

        if (method === "DELETE") {
            // Destructive: Needs Setup BEFORE it (unless it's the very first item which we handled? No, always safer to setup before delete to ensure it exists)
            // Actually, if we just did a Setup, we are good.
            // If the PREVIOUS item was a DELETE, we definitely need a Setup.
            // If the previous item was a GET, we *might* be good, but to be "proper", let's be explicit.

            // Optimization: If the very last item added was a Setup, don't add another.
            const lastItem = newItems[newItems.length - 1];
            if (!lastItem.name.startsWith("Setup Imposter")) {
                newItems.push(createSetupItem(imposterPort));
            }
            newItems.push(item);
        } else {
            // Read-only (GET, etc): Just add it.
            // But if the previous item was a DELETE, we need a Setup!
            const lastItem = newItems[newItems.length - 1];
            if (lastItem.request && lastItem.request.method === "DELETE") {
                newItems.push(createSetupItem(imposterPort));
            }
            newItems.push(item);
        }
    });

    folder.item = newItems;
}

function main() {
    const filePath = "mountebank_postman_collection.json";

    try {
        const content = fs.readFileSync(filePath, 'utf8');
        const collection = JSON.parse(content);

        // Target specifically the "Parameter Coverage Tests" folder
        const coverageFolder = collection.item.find(i => i.name === "Parameter Coverage Tests");

        if (coverageFolder) {
            console.log("Processing Parameter Coverage Tests folder...");
            processFolder(coverageFolder);
        } else {
            console.warn("Parameter Coverage Tests folder not found!");
        }

        // Optional: Check other folders?
        // "Imposters" and "Stubs" might need logic but they seem manually curated and likely sequential.
        // User said "in each folder", suggesting maybe check others.
        // Let's aggressively ensure 'Imposters' and 'Stubs' start with a Setup too if they don't?
        // But 'Imposters' folder starts with 'Create Imposter' which IS a setup.
        // 'Stubs' folder starts with 'Setup Imposter for Stubs'.
        // So those are likely fine. The issue is my generated folder doing multiple DELETEs.

        fs.writeFileSync(filePath, JSON.stringify(collection, null, 4));
        console.log("Successfully restructured Postman tests.");
    } catch (e) {
        console.error(e);
        process.exit(1);
    }
}

main();
