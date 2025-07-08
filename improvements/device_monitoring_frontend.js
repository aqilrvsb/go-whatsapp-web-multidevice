// Add this to your dashboard.html to show real-time device connection status

// Update the loadDevices function
function loadDevices(silent = false) {
    // Check real-time connection status
    fetch('/api/devices/check-connection', { credentials: 'include' })
        .then(response => {
            if (!response.ok) {
                console.log('Check-connection failed, loading devices normally...');
                return null;
            }
            return response.json();
        })
        .then(connectionData => {
            // If we got connection data, update devices with real-time status
            if (connectionData && connectionData.code === 'SUCCESS' && connectionData.results) {
                updateDeviceConnectionStatus(connectionData.results);
            }
            
            // Now load devices as usual
            return fetch('/api/devices', { credentials: 'include' });
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to fetch devices');
            }
            return response.json();
        })
        .then(data => {
            if (data.code === 'SUCCESS' && Array.isArray(data.results)) {
                devices = data.results;
            } else {
                devices = [];
            }
            renderDevices();
        })
        .catch(error => {
            if (!silent) {
                console.error('Error loading devices:', error);
            }
        });
}

// Function to update device connection status
function updateDeviceConnectionStatus(statusResults) {
    if (!Array.isArray(statusResults)) return;
    
    statusResults.forEach(status => {
        // Find device in our devices array
        const device = devices.find(d => d.id === status.device_id);
        if (device) {
            // Update with real-time status
            device.previous_status = device.status;
            device.status = status.current_status;
            device.is_whatsapp_connected = status.is_whatsapp_connected;
            device.connection_details = status.connection_details;
            device.last_checked = new Date().toISOString();
            
            // Log status changes
            if (status.status_changed) {
                console.log(`Device ${device.name} status changed: ${status.previous_status} -> ${status.current_status}`);
            }
        }
    });
}

// Add auto-refresh for device status every 30 seconds
let deviceStatusInterval;

function startDeviceStatusMonitoring() {
    // Clear existing interval if any
    if (deviceStatusInterval) {
        clearInterval(deviceStatusInterval);
    }
    
    // Check status every 30 seconds
    deviceStatusInterval = setInterval(() => {
        if (currentView === 'devices') {
            loadDevices(true); // Silent refresh
        }
    }, 30000);
}

// Update renderDevices to show connection details
function renderDevices() {
    const devicesGrid = document.getElementById('devicesGrid');
    if (!devicesGrid) return;

    devicesGrid.innerHTML = devices.map(device => {
        // Determine status color and icon
        let statusColor = 'gray';
        let statusIcon = '○';
        let statusText = device.status || 'Unknown';
        
        switch(device.status) {
            case 'online':
            case 'connected':
                statusColor = '#4CAF50';
                statusIcon = '●';
                statusText = 'Online';
                break;
            case 'disconnected':
                statusColor = '#FF9800';
                statusIcon = '◐';
                statusText = 'Disconnected';
                break;
            case 'offline':
                statusColor = '#f44336';
                statusIcon = '○';
                statusText = 'Offline';
                break;
            case 'logged_out':
                statusColor = '#9E9E9E';
                statusIcon = '⭘';
                statusText = 'Logged Out';
                break;
            case 'not_initialized':
                statusColor = '#607D8B';
                statusIcon = '□';
                statusText = 'Not Initialized';
                break;
        }
        
        // Show connection details if available
        let connectionInfo = '';
        if (device.connection_details) {
            if (device.connection_details.phone) {
                connectionInfo = `<div class="text-xs text-gray-500 mt-1">Phone: ${device.connection_details.phone}</div>`;
            }
            if (device.connection_details.error) {
                connectionInfo += `<div class="text-xs text-red-500 mt-1">${device.connection_details.error}</div>`;
            }
        }
        
        // Show last check time
        let lastChecked = '';
        if (device.last_checked) {
            const checkTime = new Date(device.last_checked);
            const now = new Date();
            const diffSeconds = Math.floor((now - checkTime) / 1000);
            if (diffSeconds < 60) {
                lastChecked = `<div class="text-xs text-gray-400">Checked: ${diffSeconds}s ago</div>`;
            }
        }

        return `
            <div class="bg-white p-6 rounded-lg shadow-md device-card" data-device-id="${device.id}">
                <div class="flex justify-between items-start mb-4">
                    <div>
                        <h3 class="text-lg font-semibold">${device.name}</h3>
                        <p class="text-sm text-gray-600">${device.phone || 'No phone'}</p>
                    </div>
                    <div class="flex items-center">
                        <span style="color: ${statusColor}; font-size: 20px; margin-right: 5px;">${statusIcon}</span>
                        <span class="text-sm font-medium" style="color: ${statusColor}">${statusText}</span>
                    </div>
                </div>
                
                ${connectionInfo}
                ${lastChecked}
                
                <div class="mt-4 flex space-x-2">
                    ${device.status === 'logged_out' || device.status === 'not_initialized' ? 
                        `<button onclick="scanQR('${device.id}')" class="px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm">
                            Scan QR
                        </button>` : 
                        `<button onclick="refreshDevice('${device.id}')" class="px-3 py-1 bg-green-500 text-white rounded hover:bg-green-600 text-sm">
                            Refresh
                        </button>`
                    }
                    ${device.status === 'disconnected' ? 
                        `<button onclick="reconnectDevice('${device.id}')" class="px-3 py-1 bg-orange-500 text-white rounded hover:bg-orange-600 text-sm">
                            Reconnect
                        </button>` : ''
                    }
                    <button onclick="logoutDevice('${device.id}')" class="px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600 text-sm">
                        Logout
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// Add reconnect function
function reconnectDevice(deviceId) {
    showNotification('Attempting to reconnect device...', 'info');
    
    fetch('/api/devices/reconnect-all', {
        method: 'POST',
        credentials: 'include'
    })
    .then(response => response.json())
    .then(data => {
        if (data.code === 'SUCCESS') {
            showNotification('Reconnection attempted', 'success');
            // Reload devices to show updated status
            setTimeout(() => loadDevices(), 2000);
        } else {
            showNotification('Reconnection failed', 'error');
        }
    })
    .catch(error => {
        console.error('Reconnect error:', error);
        showNotification('Reconnection error', 'error');
    });
}

// Start monitoring when page loads
document.addEventListener('DOMContentLoaded', function() {
    startDeviceStatusMonitoring();
});

// Clean up when page unloads
window.addEventListener('beforeunload', function() {
    if (deviceStatusInterval) {
        clearInterval(deviceStatusInterval);
    }
});