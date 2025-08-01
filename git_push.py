import subprocess
import os

os.chdir(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main')

try:
    # Try to add all files
    subprocess.run(['git', 'add', '-A'], check=True)
    print("Files added successfully")
    
    # Try to commit
    subprocess.run(['git', 'commit', '-m', 'Fix sequence device report - step_order column issue'], check=True)
    print("Committed successfully")
    
    # Try to push
    subprocess.run(['git', 'push', '--force', 'origin', 'main'], check=True)
    print("Pushed to GitHub successfully!")
    
except subprocess.CalledProcessError as e:
    print(f"Git command failed: {e}")
except FileNotFoundError:
    print("Git not found in PATH")
