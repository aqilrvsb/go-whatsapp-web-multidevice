package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// DeviceMonitorHandler provides HTTP endpoints for device monitoring
type DeviceMonitorHandler struct {
	deviceManager *DeviceManager
}

// NewDeviceMonitorHandler creates a new monitor handler
func NewDeviceMonitorHandler(dm *DeviceManager) *DeviceMonitorHandler {
	return &DeviceMonitorHandler{
		deviceManager: dm,
	}
}

// RegisterRoutes registers monitoring routes
func (h *DeviceMonitorHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/devices/stats", h.GetAllDeviceStats).Methods("GET")
	router.HandleFunc("/api/devices/{deviceId}/stats", h.GetDeviceStats).Methods("GET")
	router.HandleFunc("/api/devices/{deviceId}/reset", h.ResetDevice).Methods("POST")
	router.HandleFunc("/api/devices/monitor", h.GetMonitorDashboard).Methods("GET")
}

// GetAllDeviceStats returns stats for all devices
func (h *DeviceMonitorHandler) GetAllDeviceStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	stats, err := h.deviceManager.GetAllDeviceStats(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetDeviceStats returns stats for a specific device
func (h *DeviceMonitorHandler) GetDeviceStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	
	ctx := context.Background()
	stats, err := h.deviceManager.GetDeviceStats(ctx, deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ResetDevice resets counters for a device
func (h *DeviceMonitorHandler) ResetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	
	ctx := context.Background()
	err := h.deviceManager.ResetDeviceCounters(ctx, deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Device counters reset",
	})
}

// GetMonitorDashboard returns HTML dashboard
func (h *DeviceMonitorHandler) GetMonitorDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Device Monitor Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .device-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 20px; }
        .device-card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            position: relative;
        }
        .device-id { font-weight: bold; color: #333; margin-bottom: 10px; }
        .stats { margin: 10px 0; }
        .stat-row { display: flex; justify-content: space-between; margin: 5px 0; }
        .progress-bar {
            width: 100%;
            height: 20px;
            background: #e0e0e0;
            border-radius: 10px;
            overflow: hidden;
            margin: 5px 0;
        }
        .progress-fill {
            height: 100%;
            background: #4CAF50;
            transition: width 0.3s ease;
        }
        .progress-fill.warning { background: #ff9800; }
        .progress-fill.danger { background: #f44336; }
        .locked { position: absolute; top: 10px; right: 10px; color: #f44336; }
        .refresh-btn {
            background: #2196F3;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            margin: 20px 0;
        }
        .header { 
            background: white; 
            padding: 20px; 
            border-radius: 8px; 
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { margin: 0 0 10px 0; color: #333; }
        .subtitle { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Device Monitor Dashboard</h1>
            <p class="subtitle">Real-time Redis-based device tracking (No race conditions!)</p>
            <button class="refresh-btn" onclick="loadStats()">🔄 Refresh Stats</button>
        </div>
        <div id="deviceGrid" class="device-grid">
            Loading devices...
        </div>
    </div>

    <script>
        async function loadStats() {
            try {
                const response = await fetch('/api/devices/stats');
                const stats = await response.json();
                
                const grid = document.getElementById('deviceGrid');
                grid.innerHTML = '';
                
                Object.entries(stats).forEach(([deviceId, stat]) => {
                    const hourPercent = (stat.MessagesHour / 80) * 100;
                    const dayPercent = (stat.MessagesToday / 800) * 100;
                    
                    const hourClass = hourPercent > 90 ? 'danger' : hourPercent > 70 ? 'warning' : '';
                    const dayClass = dayPercent > 90 ? 'danger' : dayPercent > 70 ? 'warning' : '';
                    
                    const card = document.createElement('div');
                    card.className = 'device-card';
                    card.innerHTML = ` + "`" + `
                        ${stat.IsLocked ? '<span class="locked">🔒 LOCKED</span>' : ''}
                        <div class="device-id">Device: ${deviceId.substring(0, 8)}...</div>
                        <div class="stats">
                            <div class="stat-row">
                                <span>Hourly:</span>
                                <span>${stat.MessagesHour}/80</span>
                            </div>
                            <div class="progress-bar">
                                <div class="progress-fill ${hourClass}" style="width: ${hourPercent}%"></div>
                            </div>
                            <div class="stat-row">
                                <span>Daily:</span>
                                <span>${stat.MessagesToday}/800</span>
                            </div>
                            <div class="progress-bar">
                                <div class="progress-fill ${dayClass}" style="width: ${dayPercent}%"></div>
                            </div>
                            <div class="stat-row" style="margin-top: 10px; font-size: 0.9em; color: #666;">
                                <span>Last Update:</span>
                                <span>${new Date(stat.Timestamp).toLocaleTimeString()}</span>
                            </div>
                        </div>
                    ` + "`" + `;
                    grid.appendChild(card);
                });
            } catch (error) {
                console.error('Failed to load stats:', error);
            }
        }
        
        // Load stats on page load
        loadStats();
        
        // Auto-refresh every 5 seconds
        setInterval(loadStats, 5000);
    </script>
</body>
</html>
	`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}