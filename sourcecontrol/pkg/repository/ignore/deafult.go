package ignore

const DefaultIgnore = `# Source Control Ignore File
# This file specifies intentionally untracked files to ignore

# Dependencies
node_modules/
bower_components/
vendor/

# Build outputs
dist/
build/
out/
*.exe
*.dll
*.so
*.dylib

# IDE files
.vscode/
.idea/
*.sublime-*
.project
.classpath
.settings/

# OS files
.DS_Store
Thumbs.db
desktop.ini

# Temporary files
*.tmp
*.temp
*.swp
*.swo
*~
.cache/

# Logs
*.log
logs/
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Environment files
.env
.env.local
.env.*.local

# Test coverage
coverage/
.nyc_output/
*.lcov

# Package files
*.tgz
*.tar.gz
*.zip
*.rar

# Python
__pycache__/
*.py[cod]
*$py.class
.Python
venv/
env/

# Java
*.class
*.jar
target/

# Ruby
*.gem
.bundle/

# Rust
target/
Cargo.lock

# Go
*.test
*.out

# Database
*.sqlite
*.sqlite3
*.db

# Secrets and keys
*.key
*.pem
*.p12
*.pfx
private/
secrets/
`
