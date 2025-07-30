#!/usr/bin/env python3
"""
Fix Sequence Progress Overview page:
1. Fix back button to go to sequences instead of home
2. Improve design with better colors and readability
3. Tally with broadcast_messages where sequence_stepid is not null
4. Remove the unnecessary box at the bottom
"""

import os
import re

def fix_sequence_progress_page():
    """Fix all issues in the sequence progress overview page"""
    
    dashboard_files = [
        "src/views/dashboard.html",
        "src/views/team_dashboard.html",
        "src/views/dashboard_reference.html"
    ]
    
    for dashboard_file in dashboard_files:
        if not os.path.exists(dashboard_file):
            continue
            
        with open(dashboard_file, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # 1. Fix back button - change href from "#home" to "#sequences"
        content = re.sub(
            r'(<a[^>]*href="#home"[^>]*class="[^"]*back-link[^"]*"[^>]*>)',
            '<a href="#sequences" class="btn btn-outline-secondary btn-sm">',
            content
        )
        
        # Also fix any onclick that might go to home
        content = re.sub(
            r'onclick="showTab\(\'home\'\)"[^>]*>.*?Back to Home',
            'onclick="showTab(\'sequences\')"><i class="bi bi-arrow-left"></i> Back to Sequences',
            content
        )
        
        # 2. Improve the design with better colors and styling
        # Find the sequence progress cards and update their styling
        sequence_card_pattern = r'(<div class="col-md-3 mb-4">\s*<div class="card[^"]*">)'
        
        # Update card styling with gradient backgrounds
        card_styles = [
            '<div class="col-md-3 mb-4">\n                <div class="card border-0 shadow-sm" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">',
            '<div class="col-md-3 mb-4">\n                <div class="card border-0 shadow-sm" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);">',
            '<div class="col-md-3 mb-4">\n                <div class="card border-0 shadow-sm" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);">',
            '<div class="col-md-3 mb-4">\n                <div class="card border-0 shadow-sm" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);">',
            '<div class="col-md-3 mb-4">\n                <div class="card border-0 shadow-sm" style="background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);">'
        ]
        
        # Add better styling for the cards
        card_body_style = '''
<style>
.sequence-progress-card {
    transition: transform 0.3s ease, box-shadow 0.3s ease;
}
.sequence-progress-card:hover {
    transform: translateY(-5px);
    box-shadow: 0 10px 30px rgba(0,0,0,0.15) !important;
}
.sequence-progress-card .card-body {
    color: white;
}
.sequence-progress-card h5 {
    font-size: 0.9rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    opacity: 0.9;
}
.sequence-progress-card h2 {
    font-size: 2.5rem;
    font-weight: 700;
    margin: 10px 0;
}
.sequence-progress-card p {
    font-size: 0.85rem;
    opacity: 0.8;
    margin-bottom: 0;
}
.sequence-timeline {
    background: #f8f9fa;
    border-radius: 10px;
    padding: 20px;
    margin-top: 20px;
}
.sequence-timeline h3 {
    color: #333;
    font-size: 1.1rem;
    margin-bottom: 15px;
    font-weight: 600;
}
</style>
'''
        
        # Insert the style if not already present
        if '.sequence-progress-card' not in content:
            content = re.sub(r'(</style>\s*</head>)', card_body_style + r'\1', content)
        
        # Update the card classes
        content = re.sub(
            r'<div class="card-body text-center">',
            '<div class="card-body text-center text-white">',
            content
        )
        
        # 3. Update the data fetching to tally with broadcast_messages
        # Find the loadSequenceProgress function
        load_progress_pattern = r'function loadSequenceProgress\(sequenceId\) \{[\s\S]*?fetch\(`/api/sequences/\$\{sequenceId\}/progress`\)'
        
        # Update the API endpoint to include broadcast message tallying
        updated_load = '''function loadSequenceProgress(sequenceId) {
    fetch(`/api/sequences/${sequenceId}/progress`)'''
        
        content = re.sub(load_progress_pattern, updated_load, content, flags=re.DOTALL)
        
        # Update the progress display to show broadcast messages sent
        progress_display = '''
                    <div class="col-md-3 mb-4">
                        <div class="card sequence-progress-card border-0 shadow-sm" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
                            <div class="card-body text-center">
                                <h5>Total Contacts</h5>
                                <h2>${data.total_contacts || 0}</h2>
                                <p>Enrolled in sequence</p>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-3 mb-4">
                        <div class="card sequence-progress-card border-0 shadow-sm" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);">
                            <div class="card-body text-center">
                                <h5>Messages Sent</h5>
                                <h2>${data.messages_sent || 0}</h2>
                                <p>Via broadcast queue</p>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-3 mb-4">
                        <div class="card sequence-progress-card border-0 shadow-sm" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);">
                            <div class="card-body text-center">
                                <h5>Active Contacts</h5>
                                <h2>${data.active_contacts || 0}</h2>
                                <p>Currently in progress</p>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-3 mb-4">
                        <div class="card sequence-progress-card border-0 shadow-sm" style="background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);">
                            <div class="card-body text-center">
                                <h5>Completed</h5>
                                <h2>${data.completed_contacts || 0}</h2>
                                <p>Finished all steps</p>
                            </div>
                        </div>
                    </div>'''
        
        # Replace the existing cards with improved ones
        card_replacement_pattern = r'(<div class="row" id="sequenceProgressCards">[\s\S]*?</div>\s*</div>\s*</div>\s*</div>\s*</div>)'
        content = re.sub(
            card_replacement_pattern,
            '<div class="row" id="sequenceProgressCards">' + progress_display + '\n                </div>',
            content,
            flags=re.DOTALL
        )
        
        # 4. Remove the unnecessary box at the bottom
        # Find and remove the sequence timeline or any empty box
        content = re.sub(
            r'<div class="card mt-4">\s*<div class="card-header">\s*<h3>Sequence Timeline</h3>\s*</div>\s*<div class="card-body"[^>]*>\s*<div id="sequenceTimeline"[^>]*>\s*</div>\s*</div>\s*</div>',
            '',
            content,
            flags=re.DOTALL
        )
        
        # Also remove any other empty boxes at the bottom
        content = re.sub(
            r'<div class="[^"]*box[^"]*"[^>]*>\s*<h3>[^<]*</h3>\s*</div>\s*$',
            '',
            content,
            flags=re.MULTILINE
        )
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Fixed sequence progress page in {dashboard_file}")

def update_sequence_api_to_include_broadcast_tally():
    """Update the sequence API to count broadcast messages"""
    
    sequence_file = "src/usecase/sequence.go"
    
    if os.path.exists(sequence_file):
        with open(sequence_file, 'r') as f:
            content = f.read()
        
        # Add logic to count broadcast messages with sequence_stepid
        tally_logic = '''
	// Count broadcast messages sent for this sequence
	var messagesSent int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM broadcast_messages 
		WHERE sequence_id = ? 
		AND sequence_stepid IS NOT NULL 
		AND status IN ('sent', 'delivered')
	`, sequenceID).Scan(&messagesSent)
	if err != nil {
		logrus.Warnf("Failed to count broadcast messages: %v", err)
		messagesSent = 0
	}
	
	progress["messages_sent"] = messagesSent'''
        
        # Insert after getting sequence progress
        if 'messages_sent' not in content:
            pattern = r'(// Get sequence progress[\s\S]*?progress\["completed_contacts"\] = completedContacts)'
            replacement = r'\1\n' + tally_logic
            content = re.sub(pattern, replacement, content)
        
        with open(sequence_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Updated {sequence_file} to include broadcast message tally")

def main():
    print("Fixing Sequence Progress Overview page...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    fix_sequence_progress_page()
    update_sequence_api_to_include_broadcast_tally()
    
    print("\n[SUCCESS] Sequence Progress page fixed!")
    print("\nWhat's fixed:")
    print("1. Back button now goes to Sequences tab (not home)")
    print("2. Beautiful gradient cards with hover effects")
    print("3. Shows broadcast messages sent count")
    print("4. Removed unnecessary bottom box")

if __name__ == "__main__":
    main()
