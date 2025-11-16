# Setup Instructions

Follow these steps to set up the SourceControl CLI application:

## Prerequisites

- **Node.js** >= 16.0.0
- **npm** >= 7.0.0
- **Git** (for source control operations)

## Quick Setup

### For Windows (PowerShell)
```powershell
# Run the setup script
.\scripts\setup.ps1
```

### For Unix/Linux/macOS
```bash
# Make scripts executable
chmod +x scripts/*.sh

# Run the setup script
./scripts/build.sh
```

### Manual Setup

1. **Install Dependencies**
   ```bash
   npm install
   ```

2. **Build the Project**
   ```bash
   npm run build
   ```

3. **Run Tests** (optional)
   ```bash
   npm test
   ```

4. **Test the CLI**
   ```bash
   node dist/cli.js --help
   ```

## Installation Options

### Global Installation
Install the CLI globally to use it anywhere:
```bash
npm install -g .
```

After global installation, you can use:
```bash
sourcecontrol --help
# or
sc --help
```

### Local Development
For development, use the development script:
```bash
npm run dev
```

This will:
- Build the project in watch mode
- Automatically restart when files change

## Verification

Test that everything is working:

```bash
# Check version
node dist/cli.js --version

# Test basic functionality
node dist/cli.js --help

# Test a command
node dist/cli.js status
```

## Development Commands

```bash
# Build project
npm run build

# Build with watch mode
npm run build:watch

# Run in development mode
npm run dev

# Run tests
npm test

# Run tests with watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Lint code
npm run lint

# Fix linting issues
npm run lint:fix

# Format code
npm run format

# Check formatting
npm run format:check

# Clean build directory
npm run clean
```

## Project Structure

```
SourceControl/
├── src/                    # TypeScript source files
│   ├── cli.ts             # CLI entry point
│   ├── index.ts           # Library exports
│   ├── types/             # Type definitions
│   ├── utils/             # Utility functions
│   ├── commands/          # CLI commands
│   └── __tests__/         # Test files
├── dist/                  # Compiled JavaScript (generated)
├── scripts/               # Build and setup scripts
├── package.json           # Dependencies and scripts
├── tsconfig.json          # TypeScript configuration
├── jest.config.js         # Jest test configuration
├── .eslintrc.js          # ESLint configuration
├── .prettierrc           # Prettier configuration
└── README.md             # Project documentation
```

## Troubleshooting

### Common Issues

1. **Node.js version mismatch**
   - Ensure you're using Node.js >= 16.0.0
   - Check with: `node --version`

2. **Permission errors (Unix/Linux/macOS)**
   - Make scripts executable: `chmod +x scripts/*.sh`

3. **Build errors**
   - Clear node_modules: `rm -rf node_modules package-lock.json`
   - Reinstall: `npm install`

4. **TypeScript errors**
   - Check TypeScript version: `npx tsc --version`
   - Run type check: `npx tsc --noEmit`

### Getting Help

If you encounter issues:

1. Check the logs in the terminal
2. Run with verbose flag: `node dist/cli.js --verbose`
3. Check the issue tracker
4. Ensure all dependencies are installed correctly

## Next Steps

After setup is complete:

1. Read the [README.md](README.md) for usage instructions
2. Try the interactive commands like `sc init`
3. Explore the available commands with `sc --help`
4. Configure your settings with the config file

Happy coding! 🚀