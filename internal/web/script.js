document.addEventListener('DOMContentLoaded', () => {
    fetchImposters();
    fetchConfig();

    document.getElementById('refreshBtn').addEventListener('click', fetchImposters);
});

async function fetchImposters() {
    const container = document.getElementById('impostersList');
    container.innerHTML = '<div class="loading">Loading...</div>';

    try {
        const response = await fetch('/imposters');
        const data = await response.json();
        
        renderImposters(data.imposters);
    } catch (error) {
        container.innerHTML = `<div class="error">Failed to load imposters: ${error.message}</div>`;
    }
}

async function fetchConfig() {
    try {
        const response = await fetch('/config');
        const data = await response.json();
        
        const container = document.getElementById('serverConfig');
        container.innerHTML = `<pre>${JSON.stringify(data, null, 2)}</pre>`;
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

function renderImposters(imposters) {
    const container = document.getElementById('impostersList');
    
    if (!imposters || imposters.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <h3>No Active Imposters</h3>
                <p>Create an imposter using the API or CLI to see it here.</p>
            </div>
        `;
        return;
    }

    container.innerHTML = imposters.map(imposter => `
        <div class="card">
            <h3>
                ${imposter.name || 'Imposter'} 
                <span class="badge">${imposter.protocol.toUpperCase()}</span>
            </h3>
            <div class="card-meta">
                Port: <strong>${imposter.port}</strong>
            </div>
            <div class="card-stats">
                <div class="stat-item">
                    <span class="stat-label">Requests</span>
                    <span class="stat-value">${imposter.numberOfRequests}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Stubs</span>
                    <span class="stat-value">${imposter.stubs ? imposter.stubs.length : 0}</span>
                </div>
            </div>
        </div>
    `).join('');
}
