# CASL2 VS Code Extension Development Guide

## Setup

### Prerequisites

- Node.js 18+ and pnpm installed
- VS Code 1.75.0 or higher
- c2c2 executable with DAP support

### Installation

1. Navigate to the extension directory:
```bash
cd vscode-casl2-debug
```

2. Install dependencies using pnpm:
```bash
pnpm install
```

3. Compile the TypeScript code:
```bash
pnpm run compile
```

## Development

### Running the Extension

1. Open the `vscode-casl2-debug` folder in VS Code
2. Press `F5` to launch the Extension Development Host
3. A new VS Code window will open with the extension loaded
4. Open a CASL2 file (`.cas`) and press `F5` to start debugging

### Watching for Changes

Run the watch task to automatically recompile on file changes:
```bash
pnpm run watch
```

Or use VS Code's build task (Ctrl+Shift+B).

## Building

### Compile TypeScript

```bash
pnpm run compile
```

This compiles TypeScript files to JavaScript in the `out/` directory.

### Package Extension

To create a `.vsix` package:

```bash
pnpm run package
```

This creates a `casl2-debug-1.0.0.vsix` file that can be installed in VS Code.

## Testing

### Manual Testing

1. Build the extension (F5 in VS Code or `pnpm run compile`)
2. In the Extension Development Host:
   - Open a CASL2 file
   - Set breakpoints by clicking in the gutter
   - Press F5 to start debugging
   - Use debug controls (step, continue, etc.)
   - Check Variables pane for register values

### Test Program

Use the example program in `../examples/simple_add.cas`:

```casl2
MAIN    START
        LD      GR0, =10
        LD      GR1, =20
        ADDA    GR0, GR1
        ST      GR0, RESULT
        RET
RESULT  DS      1
        END
```

### Verify Features

- [ ] Syntax highlighting works
- [ ] Breakpoints can be set
- [ ] Debug session starts successfully
- [ ] Stepping through code works
- [ ] Variables show correct register values
- [ ] Continue/pause work correctly
- [ ] Debug session can be stopped

## Project Structure

```
vscode-casl2-debug/
├── src/
│   └── extension.ts          # Main extension code
├── syntaxes/
│   └── casl2.tmLanguage.json # Syntax highlighting
├── images/
│   ├── icon.svg              # Extension icon (SVG)
│   └── README.md             # Icon generation instructions
├── .vscode/
│   ├── launch.json           # Debug configuration
│   ├── tasks.json            # Build tasks
│   └── extensions.json       # Recommended extensions
├── package.json              # Extension manifest
├── tsconfig.json             # TypeScript configuration
├── language-configuration.json # Language configuration
├── README.md                 # User documentation
├── CHANGELOG.md              # Version history
└── LICENSE                   # License file
```

## Extension Features

### Debug Adapter Integration

The extension provides:
- **Configuration Provider**: Supplies default debug configurations
- **Debug Adapter Descriptor Factory**: Starts the c2c2 DAP server
- **Server Management**: Automatically starts/stops the debug server

### Language Support

- **Syntax Highlighting**: Full CASL2 assembly syntax
- **Language Configuration**: Comments, brackets, auto-closing pairs
- **File Association**: `.cas` files

### Debug Capabilities

- Launch configurations
- Breakpoint support
- Step execution (over, in, out)
- Variable inspection (registers)
- Stack traces

## Configuration

### Extension Settings

The extension uses launch configuration settings:

```json
{
  "program": "${file}",      // CASL2 source file
  "stopOnEntry": true,       // Stop at entry point
  "debugServer": 4711,       // DAP server port
  "c2c2Path": "c2c2"        // Path to c2c2 executable
}
```

### Default Port

The default debug server port is 4711. Change it if:
- Port is already in use
- Running multiple debug sessions
- Firewall restrictions

## Publishing

### Prerequisites

1. Create a Publisher account on [Visual Studio Marketplace](https://marketplace.visualstudio.com/)
2. Get a Personal Access Token from Azure DevOps
3. Login with vsce:
```bash
pnpm vsce login <publisher-name>
```

### Publish to Marketplace

```bash
pnpm run publish
```

Or manually:
```bash
pnpm vsce publish
```

### Version Bumping

Before publishing, update the version in `package.json`:
```bash
# Patch version (1.0.0 -> 1.0.1)
npm version patch

# Minor version (1.0.0 -> 1.1.0)
npm version minor

# Major version (1.0.0 -> 2.0.0)
npm version major
```

## Troubleshooting

### Extension doesn't activate

- Check VS Code version (must be 1.75.0+)
- Verify `out/` directory has compiled JavaScript
- Check Output panel (View → Output → Extension Host)

### c2c2 not found

- Ensure c2c2 is in PATH
- Or set full path in `c2c2Path` configuration
- Check c2c2 has `-dap` flag support

### Debug server fails to start

- Check port is not in use: `netstat -an | grep 4711`
- Verify c2c2 executable permissions
- Check firewall settings for localhost connections

### Breakpoints not working

- Ensure source file is saved
- Check file path is absolute
- Verify breakpoints are on valid code lines (not comments/blank)

## Contributing

1. Make changes to TypeScript files in `src/`
2. Test in Extension Development Host
3. Update CHANGELOG.md
4. Update version in package.json
5. Create pull request

## License

GPL-3.0 - See LICENSE file for details.
