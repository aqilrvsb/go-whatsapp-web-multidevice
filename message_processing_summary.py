import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== HOW SEQUENCES AND CAMPAIGNS ARE PROCESSED ===\n")

print("BOTH USE THE EXACT SAME SYSTEM!\n")

# Show the key query
print("1. GetPendingMessages Query:")
print("   WHERE status = 'pending'")
print("   AND scheduled_at <= NOW()")
print("   (No difference between campaign/sequence)\n")

# Check current pending messages
cursor.execute("""
    SELECT 
        COUNT(*) as total,
        SUM(CASE WHEN campaign_id IS NOT NULL THEN 1 ELSE 0 END) as campaigns,
        SUM(CASE WHEN sequence_id IS NOT NULL THEN 1 ELSE 0 END) as sequences
    FROM broadcast_messages
    WHERE status = 'pending'
""")
result = cursor.fetchone()
print(f"2. Current Pending Messages:")
print(f"   Total: {result[0]}")
print(f"   Campaigns: {result[1]}")
print(f"   Sequences: {result[2]}\n")

print("3. Processing Flow (ultra_optimized_broadcast_processor.go):")
print("   - Gets devices with pending messages")
print("   - Calls GetPendingMessagesAndLock() for each device")
print("   - Creates broadcast pools for campaigns/sequences")
print("   - Queues messages to pools using QueueMessageToBroadcast()")
print("   - Both use IDENTICAL processing!\n")

print("4. Key Code:")
print("   if msg.CampaignID != nil {")
print("       broadcastType = 'campaign'")
print("   } else if msg.SequenceID != nil {")
print("       broadcastType = 'sequence'")
print("   }")
print("   // Then both go through same queue system\n")

print("CONCLUSION: Campaigns and sequences are processed identically!")
print("The only difference is the ID field used (campaign_id vs sequence_id)")

conn.close()
