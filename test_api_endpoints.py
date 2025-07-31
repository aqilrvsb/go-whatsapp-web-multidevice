import requests
import json

# Base URL - adjust if running on different port
BASE_URL = "http://localhost:3000"

# Test worker status
print("=== TESTING WORKER STATUS ===")
try:
    response = requests.get(f"{BASE_URL}/api/workers/status", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        print(f"Status Code: {response.status_code}")
        print(f"Response: {json.dumps(data, indent=2)}")
    else:
        print(f"Error: Status Code {response.status_code}")
        print(f"Response: {response.text}")
except Exception as e:
    print(f"Connection Error: {e}")
    print("Make sure the WhatsApp server is running on port 3000")

# Test campaign summary
print("\n=== TESTING CAMPAIGN SUMMARY ===")
try:
    response = requests.get(f"{BASE_URL}/api/campaigns/summary", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        print(f"Status Code: {response.status_code}")
        if 'results' in data and data['results']:
            results = data['results']
            print(f"Total Campaigns: {results.get('campaigns', {}).get('total', 0)}")
            print(f"Broadcast Stats: {results.get('broadcast_stats', {})}")
    else:
        print(f"Error: Status Code {response.status_code}")
        print(f"Response: {response.text}")
except Exception as e:
    print(f"Connection Error: {e}")

# Test sequence summary
print("\n=== TESTING SEQUENCE SUMMARY ===")
try:
    response = requests.get(f"{BASE_URL}/api/sequences/summary", 
                          auth=('admin', 'changeme123'))
    
    if response.status_code == 200:
        data = response.json()
        print(f"Status Code: {response.status_code}")
        if 'results' in data and data['results']:
            results = data['results']
            print(f"Total Sequences: {results.get('sequences', {}).get('total', 0)}")
            print(f"Contacts: {results.get('contacts', {})}")
    else:
        print(f"Error: Status Code {response.status_code}")
        print(f"Response: {response.text}")
except Exception as e:
    print(f"Connection Error: {e}")
