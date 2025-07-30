import re

print("Fixing analytics handlers to use proper database connection...")

file_path = r'src\ui\rest\analytics_handlers.go'

with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Replace config.MysqlDSN with proper database connection
# The analytics should use the MySQL database from the repository
content = content.replace('sql.Open("mysql", config.MysqlDSN)', 'database.GetDB()')

# We need to import the database package
if 'github.com/aldinokemal/go-whatsapp-web-multidevice/database' not in content:
    # Add import after other imports
    import_section = re.search(r'import \((.*?)\)', content, re.DOTALL)
    if import_section:
        imports = import_section.group(1)
        new_imports = imports.rstrip() + '\n\t"github.com/aldinokemal/go-whatsapp-web-multidevice/database"\n'
        content = content.replace(imports, new_imports)

# Change db, err := sql.Open to just db := database.GetDB()
content = re.sub(r'db, err := sql\.Open\("mysql", config\.MysqlDSN\)', 'db := database.GetDB()', content)
content = re.sub(r'if err != nil \{\s*return c\.Status\(500\)\.JSON\(utils\.ResponseData\{[^}]+\}\)\s*\}', '', content)

# Remove the defer db.Close() since we're using a shared connection
content = re.sub(r'defer db\.Close\(\)\s*\n', '', content)

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed analytics handlers!")
