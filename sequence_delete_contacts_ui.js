// Add Delete Contact Trigger Button to Sequences UI

// Function to delete sequence contacts by status
async function deleteSequenceContacts(sequenceId, sequenceName) {
    // Show modal with status options
    const modal = `
        <div class="modal fade" id="deleteContactsModal" tabindex="-1">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">Delete Contacts - ${sequenceName}</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                    </div>
                    <div class="modal-body">
                        <p>Select which contacts to delete based on their status:</p>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="deleteStatus" id="statusPending" value="pending" checked>
                            <label class="form-check-label" for="statusPending">
                                <span class="badge bg-warning">Pending</span> - Messages waiting to be sent
                            </label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="deleteStatus" id="statusSent" value="sent">
                            <label class="form-check-label" for="statusSent">
                                <span class="badge bg-success">Sent</span> - Messages already sent
                            </label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="deleteStatus" id="statusFailed" value="failed">
                            <label class="form-check-label" for="statusFailed">
                                <span class="badge bg-danger">Failed</span> - Messages that failed to send
                            </label>
                        </div>
                        <div class="form-check mt-3">
                            <input class="form-check-input" type="radio" name="deleteStatus" id="statusAll" value="all">
                            <label class="form-check-label" for="statusAll">
                                <span class="badge bg-dark">All</span> - Delete all messages regardless of status
                            </label>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                        <button type="button" class="btn btn-danger" onclick="confirmDeleteContacts('${sequenceId}')">
                            <i class="bi bi-trash"></i> Delete Contacts
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Add modal to page if not exists
    if (!document.getElementById('deleteContactsModal')) {
        document.body.insertAdjacentHTML('beforeend', modal);
    }
    
    // Show modal
    const modalElement = new bootstrap.Modal(document.getElementById('deleteContactsModal'));
    modalElement.show();
}

// Function to confirm and execute deletion
async function confirmDeleteContacts(sequenceId) {
    const selectedStatus = document.querySelector('input[name="deleteStatus"]:checked').value;
    
    // Close modal
    bootstrap.Modal.getInstance(document.getElementById('deleteContactsModal')).hide();
    
    // Show loading
    Swal.fire({
        title: 'Deleting Contacts...',
        text: `Removing ${selectedStatus} messages from sequence`,
        allowOutsideClick: false,
        didOpen: () => {
            Swal.showLoading();
        }
    });
    
    try {
        let endpoint;
        let body = {};
        
        if (selectedStatus === 'all') {
            endpoint = `/sequence/${sequenceId}/contacts/all`;
        } else {
            endpoint = `/sequence/${sequenceId}/contacts`;
            body = { status: selectedStatus };
        }
        
        const response = await fetch(endpoint, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(body)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            Swal.fire({
                icon: 'success',
                title: 'Contacts Deleted',
                text: result.message || `Successfully deleted ${result.deleted_count} contacts`,
                showConfirmButton: true
            }).then(() => {
                // Reload sequence data
                location.reload();
            });
        } else {
            throw new Error(result.message || 'Failed to delete contacts');
        }
    } catch (error) {
        Swal.fire({
            icon: 'error',
            title: 'Delete Failed',
            text: error.message || 'Failed to delete contacts'
        });
    }
}

// Function to add delete button to sequence cards
function addDeleteContactButton() {
    // Find all sequence cards
    document.querySelectorAll('.sequence-card, .card').forEach(card => {
        // Check if card is a sequence card (has trigger info)
        const triggerElement = card.querySelector('*:contains("Trigger:")');
        if (!triggerElement) return;
        
        // Check if button already exists
        if (card.querySelector('.btn-delete-contacts')) return;
        
        // Get sequence info
        const sequenceNameElement = card.querySelector('h5, .card-title');
        const sequenceName = sequenceNameElement ? sequenceNameElement.textContent.trim() : 'Sequence';
        
        // Extract sequence ID from existing buttons or data attributes
        const viewButton = card.querySelector('a[href*="/sequence/"]');
        let sequenceId = null;
        
        if (viewButton) {
            const match = viewButton.href.match(/\/sequence\/([^\/]+)/);
            if (match) sequenceId = match[1];
        }
        
        // If no ID found, try data attribute
        if (!sequenceId) {
            sequenceId = card.dataset.sequenceId;
        }
        
        if (!sequenceId) return;
        
        // Find button container
        const buttonContainer = card.querySelector('.d-flex, .btn-group, .card-footer');
        if (!buttonContainer) return;
        
        // Create delete contacts button
        const deleteButton = document.createElement('button');
        deleteButton.className = 'btn btn-warning btn-sm btn-delete-contacts ms-2';
        deleteButton.innerHTML = '<i class="bi bi-person-x"></i> Delete Contacts';
        deleteButton.onclick = () => deleteSequenceContacts(sequenceId, sequenceName);
        
        // Add button to container
        buttonContainer.appendChild(deleteButton);
    });
}

// Initialize when page loads
document.addEventListener('DOMContentLoaded', function() {
    // Add delete buttons to existing sequences
    setTimeout(addDeleteContactButton, 1000);
    
    // Watch for dynamic content changes
    const observer = new MutationObserver(() => {
        addDeleteContactButton();
    });
    
    observer.observe(document.body, {
        childList: true,
        subtree: true
    });
});

// CSS for the button
const style = document.createElement('style');
style.textContent = `
    .btn-delete-contacts {
        background-color: #ff9800;
        border-color: #ff9800;
        color: white;
    }
    .btn-delete-contacts:hover {
        background-color: #e68900;
        border-color: #e68900;
        color: white;
    }
    .modal .form-check {
        margin-bottom: 10px;
    }
    .modal .badge {
        margin-right: 5px;
    }
`;
document.head.appendChild(style);
