#!/usr/bin/env python3
"""
Fix HTML syntax error in dashboard templates
"""

import os
import re

def fix_html_syntax_error():
    """Fix the malformed HTML input tags"""
    
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
        
        # Fix malformed input tags - find inputs that might have extra quotes or broken attributes
        # Pattern to find potentially broken input tags
        broken_input_patterns = [
            # Fix double escaped quotes
            (r'<input type=\\"url\\"', '<input type="url"'),
            (r'class=\\"form-control\\"', 'class="form-control"'),
            (r'placeholder=\\"([^"]+)\\"', r'placeholder="\1"'),
            
            # Fix inputs that might be missing closing >
            (r'<input type="url" class="form-c"', '<input type="url" class="form-control"'),
            
            # Fix any input tags that have line breaks in attributes
            (r'<input([^>]*)\n\s*([^>]*)>', r'<input\1 \2>'),
            
            # Fix specific campaign/AI campaign image inputs
            (r'<label class="form-label">Image URL \(Optional\)</label>\s*<input type="url" class="form-control" id="imageUrl"[^>]*>', 
             '''<label class="form-label">Image URL (Optional)</label>
                        <input type="url" class="form-control" id="imageUrl" placeholder="https://example.com/image.jpg">'''),
            
            (r'<label class="form-label">Image URL \(Optional\)</label>\s*<input type="url" class="form-control" id="aiImageUrl"[^>]*>',
             '''<label class="form-label">Image URL (Optional)</label>
                        <input type="url" class="form-control" id="aiImageUrl" placeholder="https://example.com/image.jpg">'''),
        ]
        
        for pattern, replacement in broken_input_patterns:
            content = re.sub(pattern, replacement, content, flags=re.MULTILINE | re.DOTALL)
        
        # Find and fix any incomplete input tags
        # Look for inputs that don't have proper closing
        incomplete_input_pattern = r'(<input[^/>]*(?<!>))$'
        content = re.sub(incomplete_input_pattern, r'\1>', content, flags=re.MULTILINE)
        
        # Fix any inputs with broken onchange attributes
        content = re.sub(
            r'onchange="validateImageUrl\([^)]+\)[^"]*',
            lambda m: m.group(0) + '"' if not m.group(0).endswith('"') else m.group(0),
            content
        )
        
        # Ensure all input tags are properly closed
        content = re.sub(
            r'(<input[^>]+)(?<!/)>',
            r'\1>',
            content
        )
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Fixed HTML syntax in {dashboard_file}")

def validate_html_structure():
    """Basic validation to check for common HTML errors"""
    
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
        
        # Check for common issues
        issues = []
        
        # Check for escaped quotes in attributes
        if '\\"' in content:
            issues.append("Found escaped quotes (\\\")")
        
        # Check for incomplete tags
        if re.search(r'<input[^>]*[^/>]$', content, re.MULTILINE):
            issues.append("Found incomplete input tags")
        
        # Check for attributes with < character
        if re.search(r'<[^>]* <[^>]*>', content):
            issues.append("Found < character inside tag attributes")
        
        if issues:
            print(f"[WARNING] {dashboard_file} has issues: {', '.join(issues)}")
        else:
            print(f"[OK] {dashboard_file} structure looks valid")

def main():
    print("Fixing HTML syntax errors...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    fix_html_syntax_error()
    validate_html_structure()
    
    print("\n[SUCCESS] HTML syntax fixed!")

if __name__ == "__main__":
    main()
