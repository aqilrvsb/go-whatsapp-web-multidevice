#!/usr/bin/env python3
"""
Switch back to using growrvsb.com for image storage
This is more reliable than Railway's ephemeral storage
"""

import os
import re

def switch_to_growrvsb_storage():
    """Update to use growrvsb.com instead of Railway storage"""
    
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
        
        # Change back to growrvsb.com
        content = content.replace(
            "fetch('/api/upload-image'",
            "fetch('http://growrvsb.com/railwayupload'"
        )
        
        # Update success message
        content = content.replace(
            'Uploaded successfully',
            'Uploaded to growrvsb.com (persistent storage)'
        )
        
        # Fix the URL handling
        old_logic = '''// Get the URL from response
        let imageUrl = data.url || data.path;
        
        // The server returns the full URL'''
        
        new_logic = '''// Get the URL from response
        let imageUrl = data.url || data.path || data.image_url;
        
        // Ensure full URL
        if (imageUrl && !imageUrl.startsWith('http')) {
            imageUrl = 'http://growrvsb.com' + (imageUrl.startsWith('/') ? '' : '/') + imageUrl;
        }'''
        
        content = content.replace(old_logic, new_logic)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Updated {dashboard_file} to use growrvsb.com")

def create_persistent_storage_guide():
    """Create a guide explaining storage options"""
    
    guide = '''# Image Storage Options for WhatsApp System

## Current Problem with Railway Storage
- Railway uses **ephemeral storage** by default
- When Railway restarts/redeploys, all uploaded images are DELETED
- This happens on:
  - Every code push
  - Every restart
  - Railway maintenance
  - Crashes

## Recommended Solution: Use growrvsb.com

Your Laravel server has persistent storage. Use it for images:

1. **Upload endpoint**: `http://growrvsb.com/railwayupload`
2. **Storage location**: `/home/admin/public_html/public/images/RAILWAY/`
3. **Access URL**: `http://growrvsb.com/public/images/RAILWAY/filename.jpg`

## Alternative Solutions:

### 1. Railway Volumes (Paid Feature)
```
- Go to Railway → Your Service → Settings → Volumes
- Add volume at: /app/statics/images/railway
- Cost: ~$5/month for 10GB
```

### 2. Cloud Storage Services
- **Cloudinary** (Free tier: 25GB)
- **AWS S3** (Pay as you go)
- **DigitalOcean Spaces** ($5/month)
- **Backblaze B2** (10GB free)

### 3. Just Use URLs
Don't upload at all - use existing image URLs:
- Upload to growrvsb.com via FTP/cPanel
- Use the URL in sequences/campaigns
- No uploads needed from WhatsApp system

## Quick Fix for Now

The code has been updated to use growrvsb.com for storage.
Make sure your Laravel endpoint returns JSON like:

```json
{
    "success": true,
    "url": "http://growrvsb.com/public/images/RAILWAY/1234567890_image.jpg"
}
```
'''
    
    with open("PERSISTENT_STORAGE_GUIDE.md", 'w') as f:
        f.write(guide)
    
    print("[OK] Created persistent storage guide")

def main():
    print("Switching to growrvsb.com for persistent image storage...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    switch_to_growrvsb_storage()
    create_persistent_storage_guide()
    
    print("\n[SUCCESS] Switched to growrvsb.com storage!")
    print("\nBenefits:")
    print("- Images won't be lost when Railway restarts")
    print("- Persistent storage on your Laravel server")
    print("- No need for Railway volumes")
    print("\nMake sure your Laravel endpoint accepts uploads!")

if __name__ == "__main__":
    main()
