# Delete Sequence Contacts Implementation

## Summary
Add a "Delete Contact Trigger" button to each sequence that allows deleting broadcast messages by status (pending/sent/failed).

## Backend Implementation

### 1. Add methods to sequence usecase (src/usecase/sequence.go):

```go
// Add these methods to the sequenceService struct

// DeleteSequenceContactsByStatus deletes broadcast messages for a sequence based on status
func (s *sequenceService) DeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error) {
    // Validate status
    validStatuses := map[string]bool{
        "pending": true,
        "sent":    true,
        "failed":  true,
    }
    
    if !validStatuses[status] {
        return 0, fmt.Errorf("invalid status: %s. Must be pending, sent, or failed", status)
    }
    
    db := database.GetDB()
    result, err := db.Exec(`
        DELETE FROM broadcast_messages 
        WHERE sequence_id = $1 AND status = $2
    `, sequenceID, status)
    
    if err != nil {
        return 0, fmt.Errorf("failed to delete: %w", err)
    }
    
    return result.RowsAffected()
}

// DeleteAllSequenceContacts deletes all broadcast messages for a sequence
func (s *sequenceService) DeleteAllSequenceContacts(sequenceID string) (int64, error) {
    db := database.GetDB()
    result, err := db.Exec(`
        DELETE FROM broadcast_messages 
        WHERE sequence_id = $1
    `, sequenceID)
    
    if err != nil {
        return 0, fmt.Errorf("failed to delete: %w", err)
    }
    
    return result.RowsAffected()
}
```

### 2. Add to sequence interface (src/domains/sequence/sequence.go):

```go
type ISequenceUsecase interface {
    // ... existing methods ...
    
    // Add these:
    DeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error)
    DeleteAllSequenceContacts(sequenceID string) (int64, error)
}
```

### 3. Add API endpoints (in your router file):

```go
// Add these routes
r.Route("/sequence/{id}", func(r chi.Router) {
    r.Delete("/contacts", func(w http.ResponseWriter, r *http.Request) {
        sequenceID := chi.URLParam(r, "id")
        
        var req struct {
            Status string `json:"status"`
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        count, err := sequenceUsecase.DeleteSequenceContactsByStatus(sequenceID, req.Status)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        json.NewEncoder(w).Encode(map[string]interface{}{
            "deleted_count": count,
            "message": fmt.Sprintf("Deleted %d messages with status %s", count, req.Status),
        })
    })
    
    r.Delete("/contacts/all", func(w http.ResponseWriter, r *http.Request) {
        sequenceID := chi.URLParam(r, "id")
        
        count, err := sequenceUsecase.DeleteAllSequenceContacts(sequenceID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        json.NewEncoder(w).Encode(map[string]interface{}{
            "deleted_count": count,
            "message": fmt.Sprintf("Deleted %d messages", count),
        })
    })
})
```

## Frontend Implementation

Add this JavaScript to your sequences page:

```javascript
// Function to show delete modal
function showDeleteContactsModal(sequenceId, sequenceName) {
    const modalHtml = `
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
                            <input class="form-check-input" type="radio" name="deleteStatus" value="pending" checked>
                            <label class="form-check-label">
                                <span class="badge bg-warning">Pending</span> - Messages waiting to be sent
                            </label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="deleteStatus" value="sent">
                            <label class="form-check-label">
                                <span class="badge bg-success">Sent</span> - Messages already sent
                            </label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="deleteStatus" value="failed">
                            <label class="form-check-label">
                                <span class="badge bg-danger">Failed</span> - Messages that failed
                            </label>
                        </div>
                        <div class="form-check mt-2">
                            <input class="form-check-input" type="radio" name="deleteStatus" value="all">
                            <label class="form-check-label">
                                <span class="badge bg-dark">All</span> - Delete all messages
                            </label>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                        <button type="button" class="btn btn-danger" onclick="deleteContacts('${sequenceId}')">
                            Delete Contacts
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Remove existing modal if any
    $('#deleteContactsModal').remove();
    $('body').append(modalHtml);
    $('#deleteContactsModal').modal('show');
}

// Function to delete contacts
async function deleteContacts(sequenceId) {
    const status = $('input[name="deleteStatus"]:checked').val();
    $('#deleteContactsModal').modal('hide');
    
    try {
        let url = `/sequence/${sequenceId}/contacts`;
        let options = {
            method: 'DELETE',
            headers: {'Content-Type': 'application/json'}
        };
        
        if (status === 'all') {
            url += '/all';
        } else {
            options.body = JSON.stringify({status: status});
        }
        
        const response = await fetch(url, options);
        const result = await response.json();
        
        if (response.ok) {
            alert(result.message);
            location.reload(); // Reload to update counts
        } else {
            alert('Error: ' + result.message);
        }
    } catch (error) {
        alert('Failed to delete contacts: ' + error.message);
    }
}
```

### Add button to each sequence card:

```html
<!-- Add this button to each sequence card -->
<button class="btn btn-warning btn-sm" onclick="showDeleteContactsModal('${sequence.id}', '${sequence.name}')">
    <i class="bi bi-person-x"></i> Delete Contacts
</button>
```

## Usage

1. Click "Delete Contacts" button on any sequence
2. Select status to delete:
   - **Pending**: Scheduled messages not yet sent
   - **Sent**: Successfully sent messages
   - **Failed**: Failed messages
   - **All**: All messages regardless of status
3. Click "Delete Contacts" to confirm

## Benefits

- Clean up pending messages before re-enrollment
- Remove failed messages to retry
- Clear test data
- Manage sequence enrollments effectively

## Notes

- This deletes from `broadcast_messages` table only
- Does not affect the sequence structure
- Useful for testing and managing enrollments
- Add proper authentication to protect these endpoints
