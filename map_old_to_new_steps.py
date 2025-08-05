import pymysql
import os

# Get MySQL connection
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
uri_parts = mysql_uri.replace('mysql://', '').split('@')
user_pass = uri_parts[0].split(':')
host_db = uri_parts[1].split('/')
host_port = host_db[0].split(':')

connection = pymysql.connect(
    host=host_port[0],
    port=int(host_port[1]),
    user=user_pass[0],
    password=user_pass[1],
    database=host_db[1],
    cursorclass=pymysql.cursors.DictCursor
)

try:
    with connection.cursor() as cursor:
        # Check both sequences
        sequences = [
            ('4fc6eb97-bfd2-4509-8d13-d444dd9a85b3', 'HOT Seqeunce'),
            ('deccef4f-8ae1-4ed6-891c-bcb7d12baa8a', 'WARM Sequence')
        ]
        
        for seq_id, seq_name in sequences:
            print(f"\n{'='*80}")
            print(f"Analyzing: {seq_name} (ID: {seq_id})")
            print('='*80)
            
            # Get current steps from sequence_steps table
            print("\nCurrent steps in sequence_steps table:")
            cursor.execute("""
                SELECT 
                    id as step_id,
                    COALESCE(day_number, day, 1) as day_num,
                    message_type,
                    content
                FROM sequence_steps
                WHERE sequence_id = %s
                ORDER BY COALESCE(day_number, day, 1)
            """, (seq_id,))
            
            current_steps = cursor.fetchall()
            current_step_map = {}
            
            for step in current_steps:
                day = step['day_num']
                content_preview = (step['content'] or '')[:50].encode('ascii', 'ignore').decode('ascii') if step['content'] else 'No content'
                print(f"  Day {day}: {step['step_id']} ({step['message_type']}) - {content_preview}...")
                current_step_map[day] = step['step_id']
            
            # Get old step IDs from broadcast_messages with their patterns
            print(f"\nOld step IDs found in broadcast_messages:")
            cursor.execute("""
                SELECT 
                    bm.sequence_stepid as old_step_id,
                    COUNT(*) as message_count,
                    MIN(bm.scheduled_at) as earliest,
                    MAX(bm.scheduled_at) as latest,
                    DATEDIFF(MIN(bm.scheduled_at), MIN(bm.created_at)) as days_after_creation
                FROM broadcast_messages bm
                WHERE bm.sequence_id = %s
                AND bm.sequence_stepid IS NOT NULL
                GROUP BY bm.sequence_stepid
                ORDER BY days_after_creation
            """, (seq_id,))
            
            old_steps = cursor.fetchall()
            old_step_map = {}
            
            for i, step in enumerate(old_steps):
                estimated_day = i + 1  # Assuming they're in order
                print(f"  Step {estimated_day} (Day {step['days_after_creation']}): {step['old_step_id']}")
                print(f"    Messages: {step['message_count']}")
                print(f"    Date range: {step['earliest']} to {step['latest']}")
                old_step_map[estimated_day] = step['old_step_id']
            
            # Create mapping
            print(f"\nMapping old step IDs to new step IDs:")
            update_map = {}
            for day in sorted(set(current_step_map.keys()) | set(old_step_map.keys())):
                if day in current_step_map and day in old_step_map:
                    old_id = old_step_map[day]
                    new_id = current_step_map[day]
                    update_map[old_id] = new_id
                    print(f"  Day {day}: {old_id} -> {new_id}")
            
            # Count messages to update
            if update_map:
                print(f"\nMessages to update:")
                for old_id, new_id in update_map.items():
                    cursor.execute("""
                        SELECT COUNT(*) as count
                        FROM broadcast_messages
                        WHERE sequence_stepid = %s
                    """, (old_id,))
                    result = cursor.fetchone()
                    print(f"  {old_id} -> {new_id}: {result['count']} messages")
                    
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
