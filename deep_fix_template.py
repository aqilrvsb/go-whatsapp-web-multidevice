#!/usr/bin/env python3
"""
Deep search for the template error
"""

import os
import re

def deep_search_error():
    """Search for all potential template errors"""
    
    dashboard_files = [
        "src/views/dashboard.html",
        "src/views/team_dashboard.html",
        "src/views/dashboard_reference.html"
    ]
    
    for dashboard_file in dashboard_files:
        if not os.path.exists(dashboard_file):
            continue
            
        print(f"\nSearching {dashboard_file}...")
        
        with open(dashboard_file, 'r', encoding='utf-8') as f:
            content = f.read()
            lines = content.split('\n')
        
        # Search for any line containing the pattern from error
        for i, line in enumerate(lines):
            # Look for escaped quotes
            if '\\"' in line:
                print(f"Line {i+1}: Found escaped quotes: {line.strip()[:100]}")
            
            # Look for the exact pattern from error
            if 'type="url" class="form-c' in line or 'type=\\"url\\" class=\\"form-c' in line:
                print(f"Line {i+1}: Found error pattern: {line.strip()[:100]}")
                
            # Look for incomplete class attributes
            if 'class="form-c' in line and 'form-control' not in line:
                print(f"Line {i+1}: Incomplete class: {line.strip()[:100]}")
            
            # Look for malformed input tags
            if '<input' in line and line.count('<') > line.count('>'):
                print(f"Line {i+1}: Unbalanced tags: {line.strip()[:100]}")
            
            # Look for template expressions that might contain HTML
            if '{{' in line and '<' in line:
                print(f"Line {i+1}: Template with HTML: {line.strip()[:100]}")

def fix_all_inputs():
    """Replace all URL inputs with clean versions"""
    
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
        
        # Fix any escaped quotes first
        content = content.replace('\\"', '"')
        
        # Replace all variations of the image URL inputs
        replacements = [
            # Fix sequence image input - any variation
            (r'<label[^>]*>Image URL[^<]*</label>[^<]*<input[^>]*id="dayMessageImageUrl"[^>]*>[^<]*(?:<small[^>]*>[^<]*</small>)?[^<]*(?:<div[^>]*id="dayImagePreview"[^>]*>[^<]*</div>)?',
             '''<label class="form-label">Image URL (Optional)</label>
                            <input type="url" class="form-control" id="dayMessageImageUrl" placeholder="https://example.com/image.jpg">
                            <small class="text-muted">Enter the full URL of your image</small>
                            <div id="dayImagePreview" class="mt-2"></div>'''),
            
            # Fix campaign image input - any variation
            (r'<label[^>]*>(?:Campaign )?Image[^<]*</label>[^<]*<input[^>]*id="imageUrl"[^>]*>[^<]*(?:<small[^>]*>[^<]*</small>)?[^<]*(?:<div[^>]*id="imagePreview"[^>]*>[^<]*</div>)?',
             '''<label class="form-label">Image URL (Optional)</label>
                    <input type="url" class="form-control" id="imageUrl" placeholder="https://example.com/image.jpg">
                    <small class="text-muted">Enter the full URL of your campaign image</small>
                    <div id="imagePreview" class="mt-2"></div>'''),
            
            # Fix AI campaign image input - any variation
            (r'<label[^>]*>(?:Campaign )?Image[^<]*</label>[^<]*<input[^>]*id="aiImageUrl"[^>]*>[^<]*(?:<small[^>]*>[^<]*</small>)?[^<]*(?:<div[^>]*id="aiImagePreview"[^>]*>[^<]*</div>)?',
             '''<label class="form-label">Image URL (Optional)</label>
                            <input type="url" class="form-control" id="aiImageUrl" placeholder="https://example.com/image.jpg">
                            <small class="text-muted">Enter the full URL of your campaign image</small>
                            <div id="aiImagePreview" class="mt-2"></div>'''),
            
            # Fix any broken input with form-c
            (r'<input[^>]*class="form-c[^"]*"[^>]*>',
             lambda m: m.group(0).replace('class="form-c', 'class="form-control')),
             
            # Remove any onchange with innerHTML
            (r'onerror="[^"]*innerHTML[^"]*"',
             'onerror="this.style.display=\'none\'"'),
        ]
        
        for pattern, replacement in replacements:
            if callable(replacement):
                content = re.sub(pattern, replacement, content, flags=re.DOTALL | re.IGNORECASE)
            else:
                content = re.sub(pattern, replacement, content, flags=re.DOTALL | re.IGNORECASE)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"\n[OK] Fixed all inputs in {dashboard_file}")

def validate_final():
    """Final validation"""
    
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
        
        # Check for problems
        problems = []
        if '\\"' in content:
            problems.append("escaped quotes")
        if 'class="form-c' in content and 'form-control' not in content:
            problems.append("incomplete form-control class")
        if 'innerHTML=' in content:
            problems.append("innerHTML in attributes")
            
        if problems:
            print(f"[WARNING] {dashboard_file} still has: {', '.join(problems)}")
        else:
            print(f"[OK] {dashboard_file} validated clean")

def main():
    print("Deep fixing template error...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    deep_search_error()
    fix_all_inputs()
    validate_final()
    
    print("\n[DONE] Template deep fix complete!")

if __name__ == "__main__":
    main()
