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
        # Mapping for HOT Sequence
        hot_sequence_map = {
            'ccb6fe8a-b454-4d9a-8104-7b55f27b69c4': '76b7c4c9-b106-4a63-af8c-01c92380c9f0',  # Day 1
            '347b29f8-c8d6-462b-a342-b5e383ad799e': '43fb9994-e38f-4bdd-998b-12181cbb260c',  # Day 2
            # Day 3 is missing in broadcast_messages, so we'll need to handle that separately
        }
        
        # Mapping for WARM Sequence
        warm_sequence_map = {
            '3ab9be84-2e02-48ed-8174-0ca299c63bb7': '0e65445c-aaa7-49fa-b0b4-b8cb098d2aed',  # Day 1
            'f7cdc297-523b-476a-b781-4262c3b3205e': '1e9732d5-3fb5-47bb-aea4-5d192a96b6ad',  # Day 2
            '98cd5957-5f7b-4ac1-b737-deaad743b7a3': '72e96e33-2169-4d72-8d2d-041eab647e53',  # Day 3
            '3a6a2964-1afc-4454-8cd6-4b1a65677c29': '3074e271-25d6-4633-b7aa-a17cb51fd763',  # Day 4
        }
        
        # Combine all mappings
        all_mappings = {**hot_sequence_map, **warm_sequence_map}
        
        print("Updating sequence_stepid in broadcast_messages...")
        print("=" * 80)
        
        total_updated = 0
        
        for old_id, new_id in all_mappings.items():
            # Update the sequence_stepid
            cursor.execute("""
                UPDATE broadcast_messages
                SET sequence_stepid = %s
                WHERE sequence_stepid = %s
            """, (new_id, old_id))
            
            updated_count = cursor.rowcount
            total_updated += updated_count
            
            print(f"Updated {updated_count} messages: {old_id} -> {new_id}")
            
        # Commit all updates
        connection.commit()
        
        print(f"\nTotal messages updated: {total_updated}")
        
        # Verify the update
        print("\n" + "=" * 80)
        print("Verifying update for both sequences...")
        
        sequences = [
            ('4fc6eb97-bfd2-4509-8d13-d444dd9a85b3', 'HOT Seqeunce'),
            ('deccef4f-8ae1-4ed6-891c-bcb7d12baa8a', 'WARM Sequence')
        ]
        
        for seq_id, seq_name in sequences:
            print(f"\n{seq_name}:")
            
            # Check if step IDs now match
            cursor.execute("""
                SELECT 
                    ss.id as step_id,
                    COALESCE(ss.day_number, ss.day, 1) as day_num,
                    COUNT(bm.id) as message_count
                FROM sequence_steps ss
                LEFT JOIN broadcast_messages bm ON bm.sequence_stepid = ss.id
                WHERE ss.sequence_id = %s
                GROUP BY ss.id, day_num
                ORDER BY day_num
            """, (seq_id,))
            
            results = cursor.fetchall()
            for row in results:
                print(f"  Day {row['day_num']}: {row['message_count']} messages")
                
except Exception as e:
    print(f"Error: {e}")
    connection.rollback()
finally:
    connection.close()
