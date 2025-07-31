import requests
import json
from datetime import datetime, timedelta

# Base URL - adjust if running on different port
BASE_URL = "http://localhost:3000"

# Test campaign summary with and without date filters
print("=== TESTING CAMPAIGN SUMMARY API ===")

# Test 1: Without date filter (should show all campaigns)
print("\n1. Testing without date filter:")
try:
    response = requests.get(f"{BASE_URL}/api/campaigns/summary", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        print(f"Status Code: {response.status_code}")
        print(f"Response Code: {data.get('code')}")
        
        if 'results' in data and data['results']:
            results = data['results']
            campaigns = results.get('campaigns', {})
            print(f"\nCampaign Statistics:")
            print(f"  Total: {campaigns.get('total', 0)}")
            print(f"  Pending: {campaigns.get('pending', 0)}")
            print(f"  Triggered: {campaigns.get('triggered', 0)}")
            print(f"  Processing: {campaigns.get('processing', 0)}")
            print(f"  Sent: {campaigns.get('sent', 0)}")
            print(f"  Failed: {campaigns.get('failed', 0)}")
            
            broadcast_stats = results.get('broadcast_stats', {})
            print(f"\nBroadcast Statistics:")
            print(f"  Should Send: {broadcast_stats.get('total_should_send', 0)}")
            print(f"  Done Send: {broadcast_stats.get('total_done_send', 0)}")
            print(f"  Failed Send: {broadcast_stats.get('total_failed_send', 0)}")
            print(f"  Remaining: {broadcast_stats.get('total_remaining_send', 0)}")
            
            recent = results.get('recent_campaigns', [])
            if recent:
                print(f"\nRecent Campaigns ({len(recent)}):")
                for camp in recent[:3]:  # Show first 3
                    print(f"  - {camp.get('title')} ({camp.get('status')})")
        else:
            print("No results in response")
            print(f"Full response: {json.dumps(data, indent=2)}")
    else:
        print(f"Error: Status Code {response.status_code}")
        print(f"Response: {response.text}")
except Exception as e:
    print(f"Connection Error: {e}")
    print("Make sure the WhatsApp server is running on port 3000")

# Test 2: With today's date filter
print("\n\n2. Testing with today's date filter:")
today = datetime.now().strftime('%Y-%m-%d')
tomorrow = (datetime.now() + timedelta(days=1)).strftime('%Y-%m-%d')

try:
    response = requests.get(f"{BASE_URL}/api/campaigns/summary?start_date={today}&end_date={tomorrow}", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        print(f"Status Code: {response.status_code}")
        
        if 'results' in data and data['results']:
            results = data['results']
            campaigns = results.get('campaigns', {})
            print(f"Today's Campaigns: Total={campaigns.get('total', 0)}, Pending={campaigns.get('pending', 0)}")
    else:
        print(f"Error: Status Code {response.status_code}")
except Exception as e:
    print(f"Connection Error: {e}")

# Test the raw campaigns API to see if campaigns exist
print("\n\n3. Testing raw campaigns API:")
try:
    response = requests.get(f"{BASE_URL}/api/campaigns", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        if 'results' in data:
            campaigns = data['results']
            print(f"Found {len(campaigns)} campaigns")
            for camp in campaigns[:5]:  # Show first 5
                print(f"  - ID: {camp.get('id')}, Title: {camp.get('title')}, Status: {camp.get('status')}")
    else:
        print(f"Error: Status Code {response.status_code}")
except Exception as e:
    print(f"Connection Error: {e}")
