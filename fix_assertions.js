
const fs = require('fs');

function processItem(item) {
    if (item.item) {
        item.item.forEach(processItem);
    } else if (item.request) {
        // Check if request is replayable=true
        const urlObj = item.request.url;
        let isReplayable = false;

        if (urlObj.query) {
            const replayableParam = urlObj.query.find(q => q.key === "replayable");
            if (replayableParam && replayableParam.value === "true") {
                isReplayable = true;
            }
        }
        // Fallback check in raw url string
        if (urlObj.raw && urlObj.raw.includes("replayable=true")) {
            isReplayable = true;
        }

        if (isReplayable) {
            // Find the test script
            const testEvent = item.event ? item.event.find(e => e.listen === "test") : null;
            if (testEvent && testEvent.script && testEvent.script.exec) {
                const exec = testEvent.script.exec;
                // Find line with property('_links')
                // "pm.expect(jsonData).to.have.property('_links');"
                // Replace .to.have with .to.not.have

                for (let i = 0; i < exec.length; i++) {
                    if (exec[i].includes(".to.have.property('_links')") && !exec[i].includes(".to.not.have")) {
                        exec[i] = exec[i].replace(".to.have.property('_links')", ".to.not.have.property('_links')");
                        console.log(`Fixed _links assertion for ${item.name}`);
                    }
                }
            }
        }
    }
}

function main() {
    const filePath = "mountebank_postman_collection.json";

    try {
        const content = fs.readFileSync(filePath, 'utf8');
        const collection = JSON.parse(content);

        collection.item.forEach(processItem);

        fs.writeFileSync(filePath, JSON.stringify(collection, null, 4));
        console.log("Successfully fixed assertions.");
    } catch (e) {
        console.error(e);
        process.exit(1);
    }
}

main();
