
const fs = require('fs');

// Helper to create a Setup Imposter Item
function createSetupItem(port) {
    return {
        "name": `Setup Imposter ${port}`,
        "request": {
            "method": "PUT",
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
                    imposters: [
                        {
                            port: parseInt(port),
                            protocol: "http"
                        }
                    ]
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
                        "pm.test(\"Setup: Status code is 200\", function () {",
                        "    pm.response.to.have.status(200);",
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
    // 1. Cleanup existing Setup items first (at this level)
    folder.item = folder.item.filter(i => !i.name.startsWith("Setup Imposter"));

    // If folder has sub-folders, recurse
    // If folder has requests, process them

    // Check if it has mixed content? Usually strict hierarchy.
    // My generator created sub-folders.

    // We will verify if items are requests or folders.
    // If any item is a Folder, we assume this is a Container Folder and just recurse on folders.
    // If items are Requests, we Apply the Setup Logic.

    const hasSubFolders = folder.item.some(i => i.item);

    if (hasSubFolders) {
        folder.item.forEach(subItem => {
            if (subItem.item) {
                processFolder(subItem);
            }
        });
        return;
    }

    // Process Requests Layer
    const newItems = [];
    const imposterPort = 4545;

    // We already filtered startItems = folder.item (which are request items now)
    const startItems = folder.item;

    // Always start with a Setup for this group/folder to ensure clean state
    newItems.push(createSetupItem(imposterPort));

    startItems.forEach(item => {
        const method = item.request ? item.request.method : "";

        if (method === "DELETE") {
            // Ensure Setup before DELETE
            // Check if last item was Setup
            const lastItem = newItems[newItems.length - 1];
            if (!lastItem.name.startsWith("Setup Imposter")) {
                newItems.push(createSetupItem(imposterPort));
            }
            newItems.push(item);

            // Note: After DELETE, we destroyed it.
            // If next item needs it, we need another Setup.
            // The loop will handle next item.
        } else {
            // Read-only (GET, etc)
            // If previous was DELETE, we need Setup data.
            const lastItem = newItems[newItems.length - 1];
            // If last item was DELETE, it definitely needs Setup.
            // If last item was GET, it persists.
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

        fs.writeFileSync(filePath, JSON.stringify(collection, null, 4));
        console.log("Successfully restructured Postman tests recursively.");
    } catch (e) {
        console.error(e);
        process.exit(1);
    }
}

main();
