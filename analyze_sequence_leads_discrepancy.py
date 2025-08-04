import os

# Let's analyze the SQL queries used in different parts of the system

print("=== ANALYZING SEQUENCE LEADS CALCULATION DISCREPANCY ===\n")

print("1. SEQUENCE SUMMARY - Total Leads Calculation:")
print("   Location: app.go lines ~2250-2300")
print("   Query for each sequence:")
print("""
   SELECT 
       COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) AS total_leads
   FROM broadcast_messages
   WHERE sequence_id = ?
   [AND DATE filters if applied]
""")

print("\n2. SEQUENCE DEVICE REPORT - Overall Total Leads:")
print("   Location: app.go lines ~4620-4650")
print("   Query for overall totals:")
print("""
   SELECT 
       COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
   FROM broadcast_messages
   WHERE sequence_id = ? 
   AND user_id = ?
   [AND DATE filters if applied]
""")

print("\n3. STEP-WISE STATISTICS - Total Leads per Step:")
print("   Location: app.go lines ~4680-4720")
print("   Query for step totals:")
print("""
   SELECT 
       sequence_stepid,
       COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
   FROM broadcast_messages
   WHERE sequence_id = ? 
   AND user_id = ?
   AND sequence_stepid IS NOT NULL
   [AND DATE filters if applied]
   GROUP BY sequence_stepid
""")

print("\n=== KEY DIFFERENCES IDENTIFIED ===\n")

print("1. AGGREGATION METHOD ISSUE:")
print("   - Sequence Summary: Counts unique phone+device for the ENTIRE sequence")
print("   - Step Statistics: Counts unique phone+device PER STEP")
print("   - Problem: If same phone+device appears in multiple steps, it's counted multiple times")
print("   - Example: If John (phone: 123) on Device A receives 3 steps:")
print("     * Step 1: John+DeviceA = 1 lead")
print("     * Step 2: John+DeviceA = 1 lead") 
print("     * Step 3: John+DeviceA = 1 lead")
print("     * Step Total Sum = 3 (WRONG!)")
print("     * Sequence Total = 1 (CORRECT!)")

print("\n2. FRONTEND CALCULATION ISSUE:")
print("   Location: dashboard.html lines ~6820-6850")
print("   The frontend sums up total_leads from each step:")
print("""
   // For total_leads, use the max value as approximation
   stepStats[step.step_id].total_leads = Math.max(
       stepStats[step.step_id].total_leads,
       step.total_leads || 0
   );
""")
print("   This is trying to use MAX but still ends up summing across steps")

print("\n3. BACKEND totalLeadsSum CALCULATION:")
print("   Location: app.go line ~2360")
print("""
   // Calculate totals from individual sequences
   for _, seq := range sequencesWithFlows {
       if leads, ok := seq["total_leads"].(int); ok {
           totalLeadsSum += leads  // This SUMS leads from each sequence
       }
   }
""")

print("\n=== SOLUTION ===\n")

print("The issue is that total_leads should NOT be summed across steps or sequences.")
print("Instead:")
print("1. For a single sequence: COUNT(DISTINCT phone+device) across ALL steps")
print("2. For multiple sequences: COUNT(DISTINCT phone+device) across ALL sequences")
print("3. For step statistics: Show leads per step but DON'T sum them")

print("\n=== SQL FIX NEEDED ===")
print("""
For accurate total leads in sequence summary:
SELECT COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
FROM broadcast_messages
WHERE user_id = ?
AND sequence_id IN (SELECT id FROM sequences WHERE user_id = ?)
[AND DATE filters]

For accurate total leads in device report:
Keep the existing query - it's correct

For step statistics display:
Show individual step leads but add a note that these overlap
""")
