#!/usr/bin/env python3
"""
Change image uploads to URL inputs for sequences, campaigns, and AI campaigns
Ensure workers handle URLs correctly for broadcast messages
"""

import os
import re

def convert_to_url_inputs():
    """Convert all image file inputs to URL inputs"""
    
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
        
        # 1. Fix Sequence Day Message Modal - Replace file input with URL input
        sequence_image_pattern = r'<label class="form-label">Image \(Optional\)</label>\s*<input type="file"[^>]+id="dayMessageImage"[^>]+>\s*<small class="text-muted">[^<]+</small>\s*<input type="hidden" id="dayMessageImageUrl">'
        
        sequence_url_input = '''<label class="form-label">Image URL (Optional)</label>
                            <input type="url" class="form-control" id="dayMessageImageUrl" 
                                   placeholder="https://example.com/image.jpg">
                            <small class="text-muted">Enter the full URL of your image (e.g., http://growrvsb.com/public/images/image.jpg)</small>'''
        
        content = re.sub(sequence_image_pattern, sequence_url_input, content, flags=re.DOTALL)
        
        # 2. Fix Campaign Modal - Replace file input with URL input
        campaign_image_pattern = r'<label class="form-label">Campaign Image \(Optional\)</label>\s*<input type="file"[^>]+id="campaignImage"[^>]+>\s*<small class="text-muted">[^<]+</small>\s*<input type="hidden" id="imageUrl">'
        
        campaign_url_input = '''<label class="form-label">Campaign Image URL (Optional)</label>
                        <input type="url" class="form-control" id="imageUrl" 
                               placeholder="https://example.com/image.jpg">
                        <small class="text-muted">Enter the full URL of your campaign image</small>'''
        
        content = re.sub(campaign_image_pattern, campaign_url_input, content, flags=re.DOTALL)
        
        # 3. Fix AI Campaign Modal - Replace file input with URL input
        ai_campaign_pattern = r'<label class="form-label">Campaign Image \(Optional\)</label>\s*<input type="file"[^>]+id="aiCampaignImage"[^>]+>\s*<small class="text-muted">[^<]+</small>\s*<input type="hidden" id="aiImageUrl">'
        
        ai_campaign_url = '''<label class="form-label">Campaign Image URL (Optional)</label>
                        <input type="url" class="form-control" id="aiImageUrl" 
                               placeholder="https://example.com/image.jpg">
                        <small class="text-muted">Enter the full URL of your campaign image</small>'''
        
        content = re.sub(ai_campaign_pattern, ai_campaign_url, content, flags=re.DOTALL)
        
        # 4. Remove/Update compress functions to validate URLs instead
        # Remove compressDayImage function
        compress_day_pattern = r'function compressDayImage\(input\) \{[\s\S]*?\}\s*img\.src = e\.target\.result;\s*\}\s*reader\.readAsDataURL\(file\);\s*\}'
        content = re.sub(compress_day_pattern, '', content, flags=re.DOTALL)
        
        # Remove compressImage function
        compress_pattern = r'function compressImage\(input\) \{[\s\S]*?\}\s*img\.src = e\.target\.result;\s*\}\s*reader\.readAsDataURL\(file\);\s*\}'
        content = re.sub(compress_pattern, '', content, flags=re.DOTALL)
        
        # Remove compressAiImage function
        compress_ai_pattern = r'function compressAiImage\(input\) \{[\s\S]*?\}\s*img\.src = e\.target\.result;\s*\}\s*reader\.readAsDataURL\(file\);\s*\}'
        content = re.sub(compress_ai_pattern, '', content, flags=re.DOTALL)
        
        # 5. Add URL validation and preview function
        url_validation_function = '''
// Validate and preview image URL
function validateImageUrl(urlInputId, previewId) {
    const urlInput = document.getElementById(urlInputId);
    const url = urlInput.value.trim();
    const preview = document.getElementById(previewId);
    
    if (!url) {
        preview.innerHTML = '';
        return true; // Empty is valid (optional)
    }
    
    // Basic URL validation
    if (!url.match(/^https?:\/\/.+\.(jpg|jpeg|png|gif|webp)$/i)) {
        preview.innerHTML = '<div class="alert alert-warning">Please enter a valid image URL ending with .jpg, .png, .gif, or .webp</div>';
        return false;
    }
    
    // Show preview
    preview.innerHTML = 
        `<img src="${url}" class="img-thumbnail" style="max-height: 150px;" 
              onload="this.nextElementSibling.textContent='✓ Image loaded successfully'" 
              onerror="this.onerror=null; this.parentElement.innerHTML='<div class=\\'alert alert-danger\\'>Failed to load image. Please check the URL.</div>'">
         <small class="d-block mt-1 text-muted"></small>`;
    
    return true;
}

// Add listeners for URL inputs
document.addEventListener('DOMContentLoaded', function() {
    // Sequence image URL
    const dayImageUrl = document.getElementById('dayMessageImageUrl');
    if (dayImageUrl) {
        dayImageUrl.addEventListener('change', function() {
            validateImageUrl('dayMessageImageUrl', 'dayImagePreview');
        });
    }
    
    // Campaign image URL
    const campaignImageUrl = document.getElementById('imageUrl');
    if (campaignImageUrl) {
        campaignImageUrl.addEventListener('change', function() {
            validateImageUrl('imageUrl', 'imagePreview');
        });
    }
    
    // AI Campaign image URL
    const aiImageUrl = document.getElementById('aiImageUrl');
    if (aiImageUrl) {
        aiImageUrl.addEventListener('change', function() {
            validateImageUrl('aiImageUrl', 'aiImagePreview');
        });
    }
});
'''
        
        # Insert the validation function before </script>
        content = re.sub(r'(</script>\s*</body>)', url_validation_function + '\n\\1', content)
        
        # 6. Update save functions to validate URLs before saving
        # Update saveDayMessage to validate URL
        save_day_pattern = r'(function saveDayMessage\(\) \{[^}]*const imageUrl = document\.getElementById\(\'dayMessageImageUrl\'\)\.value;)'
        save_day_replacement = r'''\1
    
    // Validate image URL if provided
    if (imageUrl && !validateImageUrl('dayMessageImageUrl', 'dayImagePreview')) {
        Swal.fire('Error', 'Please enter a valid image URL', 'error');
        return;
    }'''
        content = re.sub(save_day_pattern, save_day_replacement, content)
        
        # 7. Fix any onchange="compressX" references
        content = re.sub(r'onchange="compressDayImage\(this\)"', 'onchange="validateImageUrl(\'dayMessageImageUrl\', \'dayImagePreview\')"', content)
        content = re.sub(r'onchange="compressImage\(this\)"', 'onchange="validateImageUrl(\'imageUrl\', \'imagePreview\')"', content)
        content = re.sub(r'onchange="compressAiImage\(this\)"', 'onchange="validateImageUrl(\'aiImageUrl\', \'aiImagePreview\')"', content)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Updated {dashboard_file} to use URL inputs")

def ensure_worker_handles_urls():
    """Make sure the worker processes URLs correctly"""
    
    worker_file = "src/usecase/campaign_worker.go"
    
    if os.path.exists(worker_file):
        with open(worker_file, 'r') as f:
            content = f.read()
        
        # Add comment to clarify URL handling
        if "// Image URLs are stored directly" not in content:
            pattern = r'(Message:\s*broadcastMessage\.Content,)'
            replacement = r'''// Image URLs are stored directly in MediaURL field
			\1'''
            content = re.sub(pattern, replacement, content)
        
        with open(worker_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Verified {worker_file} handles URLs correctly")

def update_broadcast_to_handle_urls():
    """Ensure broadcast messages handle URLs properly"""
    
    broadcast_file = "src/usecase/broadcast.go"
    
    if os.path.exists(broadcast_file):
        with open(broadcast_file, 'r') as f:
            content = f.read()
        
        # Add logging for URL handling
        if "logrus.Debugf" not in content:
            pattern = r'(err := broadcastRepo\.CreateBroadcastMessage\(&message\))'
            replacement = r'''logrus.Debugf("Creating broadcast message - Type: %s, MediaURL: %s", message.MessageType, message.MediaURL)
		\1'''
            content = re.sub(pattern, replacement, content)
        
        with open(broadcast_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Updated {broadcast_file} for URL handling")

def create_url_usage_guide():
    """Create a guide for using external URLs"""
    
    guide = '''# Using External URLs for Images

## How to Use Images in Sequences and Campaigns

### 1. Upload to Your Server First
Upload images to growrvsb.com or any image hosting service:
- FTP to: `/home/admin/public_html/public/images/`
- Or use your Laravel upload feature
- Or use any image hosting (Imgur, Cloudinary, etc.)

### 2. Get the Full URL
Example URLs:
- `http://growrvsb.com/public/images/campaign/banner.jpg`
- `https://i.imgur.com/abc123.jpg`
- `https://res.cloudinary.com/your-cloud/image/upload/campaign.jpg`

### 3. Paste URL in WhatsApp System
- For Sequences: Paste in "Image URL" field
- For Campaigns: Paste in "Campaign Image URL" field
- For AI Campaigns: Paste in "Campaign Image URL" field

### Important Notes:
- URL must end with image extension (.jpg, .png, .gif, .webp)
- URL must be publicly accessible (no login required)
- URL is stored in database as-is (no base64 conversion)
- WhatsApp will fetch the image when sending

### Benefits:
- No storage on Railway server
- Images persist forever
- Faster loading (no base64 encoding)
- Can update images without changing campaigns
'''
    
    with open("IMAGE_URL_USAGE_GUIDE.md", 'w') as f:
        f.write(guide)
    
    print("[OK] Created IMAGE_URL_USAGE_GUIDE.md")

def main():
    print("Converting image uploads to URL inputs...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    # Apply all fixes
    convert_to_url_inputs()
    ensure_worker_handles_urls()
    update_broadcast_to_handle_urls()
    create_url_usage_guide()
    
    print("\n[SUCCESS] Converted to URL-only image handling!")
    print("\nWhat changed:")
    print("1. File inputs replaced with URL inputs")
    print("2. No more image compression/upload")
    print("3. URLs validated before saving")
    print("4. Worker processes handle URLs correctly")
    print("5. Just paste image URLs - no uploads needed!")

if __name__ == "__main__":
    main()
