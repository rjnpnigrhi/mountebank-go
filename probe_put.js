
const http = require('http');

const options = {
    hostname: 'localhost',
    port: 2525,
    path: '/imposters',
    method: 'PUT',
    headers: {
        'Content-Type': 'application/json'
    }
};

const req = http.request(options, (res) => {
    let data = '';
    res.on('data', chunk => data += chunk);
    res.on('end', () => {
        console.log(`Status: ${res.statusCode}`);
        console.log(`Body: ${data}`);
    });
});

req.write(JSON.stringify({ imposters: [] }));
req.end();
