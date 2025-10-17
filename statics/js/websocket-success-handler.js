// WebSocket Connection Success Handler
// This file ensures the QR modal closes properly when device connects

(function() {
    console.log('WebSocket success handler loaded');
    
    // Store original WebSocket onmessage handler
    let originalOnMessage = null;
    let qrModal = null;
    let connectionCheckInterval = null;
    
    // Function to close QR modal
    function closeQRModal() {
        // Try multiple methods to close modal
        const modalSelectors = [
            '#qrModal',
            '.qr-modal',
            '[data-modal="qr"]',
            '.modal.show',
            '.modal-backdrop'
        ];
        
        modalSelectors.forEach(selector => {
            const element = document.querySelector(selector);
            if (element) {
                // Try Bootstrap modal close
                if (window.bootstrap && window.bootstrap.Modal) {
                    const modal = bootstrap.Modal.getInstance(element);
                    if (modal) {
                        modal.hide();
                    }
                }
                
                // Try jQuery modal close
                if (window.$ && $.fn.modal) {
                    $(element).modal('hide');
                }
                
                // Force hide
                element.style.display = 'none';
                element.classList.remove('show');
            }
        });
        
        // Remove backdrop
        document.querySelectorAll('.modal-backdrop').forEach(backdrop => {
            backdrop.remove();
        });
        
        // Restore body scroll
        document.body.classList.remove('modal-open');
        document.body.style.overflow = '';
        
        console.log('QR modal closed');
    }
    
    // Function to show success notification
    function showSuccessNotification(message) {
        // Try to use existing notification system
        if (window.showNotification) {
            window.showNotification('success', message);
        } else if (window.toastr) {
            toastr.success(message);
        } else {
            alert(message);
        }
    }
    
    // Intercept WebSocket messages
    function interceptWebSocket() {
        const originalWebSocket = window.WebSocket;
        
        window.WebSocket = function(...args) {
            const ws = new originalWebSocket(...args);
            
            // Store original onmessage
            const originalOnMessage = ws.onmessage;
            
            // Override onmessage
            ws.onmessage = function(event) {
                try {
                    const data = JSON.parse(event.data);
                    console.log('WebSocket message received:', data);
                    
                    // Check for connection success messages
                    if (data.code === 'LOGIN_SUCCESS' || 
                        data.code === 'DEVICE_CONNECTED' || 
                        data.code === 'QR_CONNECTED' ||
                        (data.code === 'PAIR_SUCCESS' && data.result)) {
                        
                        console.log('Connection success detected:', data.code);
                        
                        // Close QR modal
                        setTimeout(() => {
                            closeQRModal();
                            showSuccessNotification(data.message || 'WhatsApp connected successfully!');
                            
                            // Reload device list after short delay
                            setTimeout(() => {
                                if (window.loadDevices) {
                                    window.loadDevices();
                                } else {
                                    location.reload();
                                }
                            }, 1000);
                        }, 500);
                    }
                    
                    // Check for logout messages
                    if (data.code === 'DEVICE_LOGGED_OUT') {
                        console.log('Device logout detected:', data.code);
                        
                        // Update device status immediately
                        if (data.result && data.result.deviceId) {
                            const deviceElement = document.querySelector(`[data-device-id="${data.result.deviceId}"]`);
                            if (deviceElement) {
                                const statusElement = deviceElement.querySelector('.device-status, .status-indicator');
                                if (statusElement) {
                                    statusElement.classList.remove('online', 'connected');
                                    statusElement.classList.add('offline', 'disconnected');
                                }
                            }
                        }
                        
                        // Reload device list
                        setTimeout(() => {
                            if (window.loadDevices) {
                                window.loadDevices();
                            }
                        }, 1000);
                    }
                } catch (e) {
                    console.error('Error parsing WebSocket message:', e);
                }
                
                // Call original handler
                if (originalOnMessage) {
                    originalOnMessage.call(this, event);
                }
            };
            
            return ws;
        };
    }
    
    // Initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', interceptWebSocket);
    } else {
        interceptWebSocket();
    }
    
    // Also check for connection status periodically when QR modal is open
    setInterval(() => {
        const qrModalVisible = document.querySelector('.modal.show, #qrModal:not([style*="display: none"])');
        if (qrModalVisible) {
            // Check if any device shows as connected
            const connectedDevice = document.querySelector('[data-status="online"], .device-status.online');
            if (connectedDevice) {
                console.log('Connected device detected, closing QR modal');
                closeQRModal();
            }
        }
    }, 2000);
})();
