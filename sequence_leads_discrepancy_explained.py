"""
SEQUENCE LEADS CALCULATION DISCREPANCY ANALYSIS
==============================================

PROBLEM: Total leads in Sequence Device Report and Step-wise Statistics 
don't match with Sequence Summary

ROOT CAUSE ANALYSIS:
-------------------

1. SEQUENCE SUMMARY (Correct Calculation):
   - Counts UNIQUE phone+device combinations across the ENTIRE sequence
   - Example: If John (123) on DeviceA gets 3 steps, he counts as 1 lead
   
2. STEP-WISE STATISTICS (Incorrect Aggregation):
   - Each step counts its unique phone+device combinations
   - Problem: Same lead appears in multiple steps
   - Example: 
     * Step 1: John+DeviceA = 1 lead
     * Step 2: John+DeviceA = 1 lead  
     * Step 3: John+DeviceA = 1 lead
     * Frontend sums these: 1+1+1 = 3 leads (WRONG!)
     * Should be: 1 unique lead across all steps

3. VISUAL EXAMPLE:
   
   Sequence "COLD Sequence" with 3 steps:
   
   Step 1 (Day 1): 
   - John (123) + Device A
   - Jane (456) + Device A
   - Bob (789) + Device B
   Total: 3 unique leads for Step 1
   
   Step 2 (Day 3):
   - John (123) + Device A (same person!)
   - Jane (456) + Device A (same person!)
   Total: 2 unique leads for Step 2
   
   Step 3 (Day 5):
   - John (123) + Device A (same person!)
   Total: 1 unique lead for Step 3
   
   CURRENT CALCULATION (WRONG):
   - Step totals: 3 + 2 + 1 = 6 leads
   
   CORRECT CALCULATION:
   - Unique across all steps: 3 leads (John, Jane, Bob)

THE SPECIFIC CODE ISSUES:
------------------------

1. Backend (app.go ~line 2360):
   ```go
   // This SUMS leads from each sequence - WRONG!
   for _, seq := range sequencesWithFlows {
       if leads, ok := seq["total_leads"].(int); ok {
           totalLeadsSum += leads
       }
   }
   ```

2. Frontend Step Statistics (dashboard.html ~line 6850):
   ```javascript
   // This tries to use MAX but still shows wrong totals
   stepStats[step.step_id].total_leads = Math.max(
       stepStats[step.step_id].total_leads,
       step.total_leads || 0
   );
   ```

3. The step statistics cards show individual step leads which get 
   visually "summed up" by users, creating confusion

SOLUTIONS:
----------

Option 1: Fix Backend Calculation (Recommended)
- For sequence summary total leads, use a single query:
  ```sql
  SELECT COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id))
  FROM broadcast_messages
  WHERE user_id = ? 
  AND sequence_id IS NOT NULL
  [AND date filters]
  ```

Option 2: Fix Frontend Display
- Show step statistics but add a note: 
  "Note: Leads may overlap between steps. Total unique leads shown above."
- Don't sum step leads in the UI

Option 3: Change Step Statistics Display
- Instead of "Total Leads" per step, show:
  - "Messages in Step" or
  - "Recipients in Step" with a note about overlaps

VERIFICATION:
------------
To verify this issue, run these queries:

1. Get total unique leads for a sequence:
   SELECT COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id))
   FROM broadcast_messages
   WHERE sequence_id = 'YOUR_SEQUENCE_ID'

2. Get sum of leads per step (wrong way):
   SELECT SUM(step_leads) FROM (
       SELECT COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as step_leads
       FROM broadcast_messages
       WHERE sequence_id = 'YOUR_SEQUENCE_ID'
       GROUP BY sequence_stepid
   ) as step_counts

These two numbers will be different if leads receive multiple steps!
"""

print(__doc__)
