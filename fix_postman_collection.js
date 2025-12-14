
const fs = require('fs');

function main() {
    const filePath = "mountebank_postman_collection.json";

    try {
        const content = fs.readFileSync(filePath, 'utf8');
        const collection = JSON.parse(content);

        // Find the "Parameter Coverage Tests" folder
        const coverageFolderIndex = collection.item.findIndex(i => i.name === "Parameter Coverage Tests");

        if (coverageFolderIndex === -1) {
            console.error("Parameter Coverage Tests folder not found");
            process.exit(1);
        }

        const coverageFolder = collection.item[coverageFolderIndex];

        // Create a "Setup Imposter" request item
        const setupItem = {
            "name": "Setup Imposter for Parameter Tests",
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
                    "raw": "{\n  \"port\": {{imposterPort}},\n  \"protocol\": \"http\"\n}"
                },
                "url": {
                    "raw": "{{baseUrl}}/imposters",
                    "host": [
                        "{{baseUrl}}"
                    ],
                    "path": [
                        "imposters"
                    ]
                }
            },
            "event": [
                {
                    "listen": "test",
                    "script": {
                        "exec": [
                            "pm.test(\"Status code is 201\", function () {",
                            "    pm.response.to.have.status(201);",
                            "});"
                        ],
                        "type": "text/javascript"
                    }
                }
            ]
        };

        // Insert Setup item at the beginning of the coverage folder items
        coverageFolder.item.unshift(setupItem);

        // Also, maybe we should move this folder to the top?
        // Or ensure it stands alone. SInce we added setup, it should be fine.

        fs.writeFileSync(filePath, JSON.stringify(collection, null, 4));
        console.log("Successfully fixed Postman collection state dependency");
    } catch (e) {
        console.error("Error fixing collection:", e);
        process.exit(1);
    }
}

main();
