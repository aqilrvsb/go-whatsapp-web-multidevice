# Fix for Sequence Flow Display Issue

## Problem
When viewing/editing an existing sequence, the sequence flow grid (Flow 1-28) is not populating with the actual sequence step data.

## Root Cause
The `viewEditSequence` function tries to populate the sequence flow from `sequence.steps`, but the sequence data loaded in the sequences list might not include the full step details.

## Solution
We need to fetch the complete sequence data including steps when viewing/editing a sequence.

## Implementation

### Option 1: Modify viewEditSequence to fetch full sequence data

Replace the current `viewEditSequence` function with:

```javascript
async function viewEditSequence(sequenceId) {
    try {
        // Show loading
        Swal.fire({
            title: 'Loading sequence...',
            allowOutsideClick: false,
            didOpen: () => {
                Swal.showLoading();
            }
        });
        
        // Fetch complete sequence data with steps
        const response = await fetch(`/api/sequences/${sequenceId}`);
        const data = await response.json();
        
        if (data.code !== 'SUCCESS' || !data.results) {
            Swal.fire('Error', 'Failed to load sequence', 'error');
            return;
        }
        
        const sequence = data.results;
        currentEditingSequenceId = sequenceId;
        
        // Populate form fields
        document.getElementById('sequenceName').value = sequence.name || '';
        document.getElementById('sequenceDescription').value = sequence.description || '';
        document.getElementById('sequenceNiche').value = sequence.niche || '';
        document.getElementById('sequenceScheduleTime').value = sequence.time_schedule || sequence.schedule_time || '09:00';
        document.getElementById('sequenceMinDelay').value = sequence.min_delay_seconds || 5;
        document.getElementById('sequenceMaxDelay').value = sequence.max_delay_seconds || 15;
        document.getElementById('sequenceTrigger').value = sequence.trigger || sequence.start_trigger || '';
        
        // Reset and populate days
        sequenceDays = {};
        if (sequence.steps && Array.isArray(sequence.steps)) {
            sequence.steps.forEach(step => {
                const dayNum = step.day_number || step.day;
                sequenceDays[dayNum] = {
                    day: dayNum,
                    day_number: dayNum,
                    trigger: step.trigger || '',
                    next_trigger: step.next_trigger || '',
                    trigger_delay_hours: step.trigger_delay_hours || 24,
                    is_entry_point: step.is_entry_point || false,
                    content: step.content || '',
                    image_url: step.image_url || step.media_url || '',
                    media_url: step.media_url || step.image_url || '',
                    message_type: step.message_type || 'text',
                    send_time: step.send_time || '09:00',
                    min_delay_seconds: step.min_delay_seconds || 5,
                    max_delay_seconds: step.max_delay_seconds || 15
                };
            });
            
            // Set global delays from first step if available
            if (sequence.steps.length > 0) {
                document.getElementById('sequenceMinDelay').value = sequence.steps[0].min_delay_seconds || 5;
                document.getElementById('sequenceMaxDelay').value = sequence.steps[0].max_delay_seconds || 15;
            }
        }
        
        // Change modal title and button text
        document.querySelector('#createSequenceModal .modal-title').textContent = 'Edit Sequence';
        document.querySelector('#createSequenceModal .modal-footer .btn-primary').textContent = 'Update Sequence';
        document.querySelector('#createSequenceModal .modal-footer .btn-primary').onclick = () => updateSequence(sequenceId);
        
        // Initialize grid with preserved data
        initializeSequenceDaysGrid(true);
        
        // Close loading and show modal
        Swal.close();
        const modal = new bootstrap.Modal(document.getElementById('createSequenceModal'));
        modal.show();
        
    } catch (error) {
        console.error('Error loading sequence:', error);
        Swal.fire('Error', 'Failed to load sequence', 'error');
    }
}
```

### Option 2: Ensure loadSequences includes steps (Backend change needed)

If you want to avoid the extra API call, you would need to modify the backend to include steps when fetching sequences list. This would require changes to the Go backend.

### Option 3: Quick Frontend Fix - Load steps when opening modal

If you want a quick fix without changing too much, you can modify the existing function to fetch steps:

```javascript
// Add this before the initializeSequenceDaysGrid call in viewEditSequence
if (!sequence.steps || sequence.steps.length === 0) {
    // Fetch steps if not already loaded
    try {
        const response = await fetch(`/api/sequences/${sequenceId}`);
        const data = await response.json();
        if (data.code === 'SUCCESS' && data.results && data.results.steps) {
            sequence.steps = data.results.steps;
        }
    } catch (error) {
        console.error('Error fetching sequence steps:', error);
    }
}
```

## Testing
1. Create a sequence with multiple steps
2. Go back to sequences list
3. Click edit on the sequence
4. Verify that the Flow boxes show the actual step data (green "Set" status for days with content)
5. Click on a flow box to verify the content is loaded correctly
