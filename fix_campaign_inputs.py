#!/usr/bin/env python3
"""
Fix campaign and AI campaign image inputs that were corrupted
"""

import os
import re

def fix_campaign_image_inputs():
    """Fix the corrupted image inputs in campaigns"""
    
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
        
        # 1. Fix regular campaign image input (line 1270)
        corrupted_campaign = r'type="file" class="form-control" id="campaignImageFile" accept="image/\*" onchange="validateImageUrl\(\'imageUrl\', \'image[^"]*'
        fixed_campaign = '''<input type="url" class="form-control" id="imageUrl" 
                               placeholder="https://example.com/image.jpg"
                               onchange="validateImageUrl('imageUrl', 'imagePreview')">'''
        
        content = re.sub(corrupted_campaign, fixed_campaign, content)
        
        # Also remove the hidden input that's no longer needed
        content = re.sub(r'<input type="hidden" id="campaignImageUrl">', '', content)
        
        # 2. Fix AI campaign image input (line 1714)
        corrupted_ai = r'type="file" class="form-control" id="aiCampaignImage" accept="image[^"]*'
        fixed_ai = '''<input type="url" class="form-control" id="aiImageUrl" 
                               placeholder="https://example.com/image.jpg"
                               onchange="validateImageUrl('aiImageUrl', 'aiImagePreview')">'''
        
        content = re.sub(corrupted_ai, fixed_ai, content)
        
        # 3. Update the label text for campaigns
        content = re.sub(
            r'<label class="form-label">Image \(Optional\)</label>\s*<input type="url"',
            '<label class="form-label">Image URL (Optional)</label>\n                            <input type="url"',
            content
        )
        
        # 4. Update the help text
        content = re.sub(
            r'<small class="text-muted">Max 5MB\. Will be compressed automatically\.</small>',
            '<small class="text-muted">Enter the full URL of your campaign image</small>',
            content
        )
        
        content = re.sub(
            r'<small class="text-muted">Max 350KB\. Will be compressed automatically\.</small>',
            '<small class="text-muted">Enter the full URL of your campaign image</small>',
            content
        )
        
        # 5. Add the AI image preview div if missing
        if 'id="aiImagePreview"' not in content:
            content = re.sub(
                r'(<small class="text-muted">Enter the full URL of your campaign image</small>)',
                r'\1\n                            <div id="aiImagePreview" class="mt-2"></div>',
                content
            )
        
        # 6. Fix the saveCampaign function to use URL instead of file
        save_campaign_pattern = r'const imageFile = document\.getElementById\(\'campaignImageFile\'\)\.files\[0\];'
        save_campaign_fix = 'const imageUrl = document.getElementById(\'imageUrl\').value;'
        content = re.sub(save_campaign_pattern, save_campaign_fix, content)
        
        # 7. Fix the saveAICampaign function
        save_ai_pattern = r'const imageFile = document\.getElementById\(\'aiCampaignImage\'\)\.files\[0\];'
        save_ai_fix = 'const imageUrl = document.getElementById(\'aiImageUrl\').value;'
        content = re.sub(save_ai_pattern, save_ai_fix, content)
        
        # 8. Remove image compression logic from saveCampaign
        compress_logic_pattern = r'// First, handle image upload if provided[\s\S]*?processImage\.then\(imageUrl => \{'
        simple_logic = '''// Use the image URL directly
        const processImage = new Promise((resolve) => {
            resolve(imageUrl || '');
        });
        
        processImage.then(imageUrl => {'''
        content = re.sub(compress_logic_pattern, simple_logic, content, flags=re.DOTALL)
        
        # 9. Fix references to imageFile checks
        content = re.sub(r'if \(imageFile\)', 'if (imageUrl)', content)
        
        # 10. Remove compressImageFile calls
        compress_call_pattern = r'compressImageFile\(imageFile, function\(compressedBlob\) \{[\s\S]*?\}\);'
        content = re.sub(compress_call_pattern, 'resolve(imageUrl);', content, flags=re.DOTALL)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Fixed campaign image inputs in {dashboard_file}")

def verify_image_url_validation():
    """Make sure the validateImageUrl function exists"""
    
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
        
        # Check if validateImageUrl function exists
        if 'function validateImageUrl' in content:
            print(f"[OK] validateImageUrl function exists in {dashboard_file}")
        else:
            print(f"[WARNING] validateImageUrl function missing in {dashboard_file}")

def main():
    print("Fixing campaign and AI campaign image inputs...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    fix_campaign_image_inputs()
    verify_image_url_validation()
    
    print("\n[SUCCESS] Campaign image inputs fixed!")
    print("\nWhat's fixed:")
    print("1. Regular campaign - now uses URL input")
    print("2. AI campaign - now uses URL input")
    print("3. Removed file upload logic")
    print("4. Uses image URLs directly")

if __name__ == "__main__":
    main()
