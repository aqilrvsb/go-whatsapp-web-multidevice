import re

print("Fixing campaign repository for MySQL compatibility...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix 1: Replace "limit" with `limit` (backticks for MySQL reserved keyword)
content = content.replace('"limit"', '`limit`')

# Fix 2: Replace QueryRow with Exec for INSERT
old_insert = """err := r.db.QueryRow(query, campaign.UserID, campaign.CampaignDate,
		campaign.Title, campaign.Niche, targetStatus, campaign.Message, campaign.ImageURL,
		campaign.TimeSchedule, campaign.MinDelaySeconds, campaign.MaxDelaySeconds, 
		campaign.Status, campaign.AI, campaign.Limit, campaign.CreatedAt, campaign.UpdatedAt).Scan(&campaign.ID)
		
	return err"""

new_insert = """result, err := r.db.Exec(query, campaign.UserID, campaign.CampaignDate,
		campaign.Title, campaign.Niche, targetStatus, campaign.Message, campaign.ImageURL,
		campaign.TimeSchedule, campaign.MinDelaySeconds, campaign.MaxDelaySeconds, 
		campaign.Status, campaign.AI, campaign.Limit, campaign.CreatedAt, campaign.UpdatedAt)
		
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	campaign.ID = int(id)
	return nil"""

content = content.replace(old_insert, new_insert)

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed campaign repository!")
