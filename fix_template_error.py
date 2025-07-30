#!/usr/bin/env python3
"""
Find and fix the specific HTML template error
"""

import os
import re

def find_and_fix_template_error():
    """Find the specific line causing the template error"""
    
    dashboard_files = [
        "src/views/dashboard.html",
        "src/views/team_dashboard.html",
        "src/views/dashboard_reference.html"
    ]
    
    for dashboard_file in dashboard_files:
        if not os.path.exists(dashboard_file):
            continue
            
        with open(dashboard_file, 'r', encoding='utf-8') as f:
            lines = f.readlines()
        
        fixed = False
        for i, line in enumerate(lines):
            # Look for lines that might have the issue
            if '<input type="url" class="form-c' in line and 'form-control' not in line:
                print(f"Found broken line {i+1}: {line.strip()}")
                lines[i] = line.replace('class="form-c"', 'class="form-control"')
                fixed = True
            
            # Look for any line with < inside attributes
            if re.search(r'="[^"]*<[^"]*"', line):
                print(f"Found < inside attribute at line {i+1}: {line.strip()}")
                # Try to fix common cases
                lines[i] = re.sub(r'="([^"]*)<([^"]*)"', r'="\1&lt;\2"', line)
                fixed = True
            
            # Look for broken validation function calls
            if 'onchange="validateImageUrl(' in line and not line.strip().endswith('">') and not line.strip().endswith('"/>'):
                print(f"Found incomplete onchange at line {i+1}: {line.strip()}")
                # Fix it
                if '">' not in line:
                    lines[i] = line.rstrip() + '">\n'
                fixed = True
            
            # Look for any malformed URLs in placeholders
            if 'placeholder=' in line:
                # Fix any < characters in placeholders
                lines[i] = re.sub(r'placeholder="([^"]*)<([^"]*)"', r'placeholder="\1\2"', line)
        
        if fixed:
            with open(dashboard_file, 'w', encoding='utf-8') as f:
                f.writelines(lines)
            print(f"[OK] Fixed template errors in {dashboard_file}")
        else:
            print(f"[INFO] No template errors found in {dashboard_file}")

def clean_image_inputs():
    """Completely rewrite the image input sections to ensure they're clean"""
    
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
        
        # Clean sequence image input
        sequence_img_pattern = r'<div class="mb-3">\s*<label class="form-label">Image URL \(Optional\)</label>\s*<input[^>]*id="dayMessageImageUrl"[^>]*>(?:\s*<small[^>]*>[^<]*</small>)?(?:\s*<div[^>]*id="dayImagePreview"[^>]*>\s*</div>)?'
        
        clean_sequence_input = '''<div class="mb-3">
                            <label class="form-label">Image URL (Optional)</label>
                            <input type="url" class="form-control" id="dayMessageImageUrl" placeholder="https://example.com/image.jpg">
                            <small class="text-muted">Enter the full URL of your image</small>
                            <div id="dayImagePreview" class="mt-2"></div>'''
        
        content = re.sub(sequence_img_pattern, clean_sequence_input, content, flags=re.DOTALL)
        
        # Clean campaign image input
        campaign_img_pattern = r'<div class="mb-3">\s*<label class="form-label">Image URL \(Optional\)</label>\s*<input[^>]*id="imageUrl"[^>]*>(?:\s*<small[^>]*>[^<]*</small>)?(?:\s*<div[^>]*id="imagePreview"[^>]*>\s*</div>)?'
        
        clean_campaign_input = '''<div class="mb-3">
                    <label class="form-label">Image URL (Optional)</label>
                    <input type="url" class="form-control" id="imageUrl" placeholder="https://example.com/image.jpg">
                    <small class="text-muted">Enter the full URL of your campaign image</small>
                    <div id="imagePreview" class="mt-2"></div>'''
        
        content = re.sub(campaign_img_pattern, clean_campaign_input, content, flags=re.DOTALL)
        
        # Clean AI campaign image input
        ai_img_pattern = r'<div class="mb-3">\s*<label class="form-label">Image URL \(Optional\)</label>\s*<input[^>]*id="aiImageUrl"[^>]*>(?:\s*<small[^>]*>[^<]*</small>)?(?:\s*<div[^>]*id="aiImagePreview"[^>]*>\s*</div>)?'
        
        clean_ai_input = '''<div class="mb-3">
                            <label class="form-label">Image URL (Optional)</label>
                            <input type="url" class="form-control" id="aiImageUrl" placeholder="https://example.com/image.jpg">
                            <small class="text-muted">Enter the full URL of your campaign image</small>
                            <div id="aiImagePreview" class="mt-2"></div>'''
        
        content = re.sub(ai_img_pattern, clean_ai_input, content, flags=re.DOTALL)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Cleaned image inputs in {dashboard_file}")

def main():
    print("Finding and fixing HTML template error...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    find_and_fix_template_error()
    clean_image_inputs()
    
    print("\n[SUCCESS] Template errors fixed!")

if __name__ == "__main__":
    main()
