# VS Code Extension for CASL2 Debugger

This directory contains the Visual Studio Code extension for debugging CASL2/COMET2 assembly programs.

## Features

- 🐛 **Full Debug Support**: Step through CASL2 code with breakpoints
- 🎨 **Syntax Highlighting**: Color-coded CASL2 assembly syntax
- 📊 **Variable Inspection**: View all registers (PC, FR, GR0-GR7, SP) in real-time
- ⚡ **Quick Start**: Press F5 to start debugging any `.cas` file
- 🔧 **Configurable**: Customize debug server port and c2c2 path

## Installation

### From VSIX Package

1. Install dependencies and build:
```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
pnpm run package
```

2. Install in VS Code:
   - Open VS Code
   - Go to Extensions view (Ctrl+Shift+X)
   - Click "..." menu → "Install from VSIX..."
   - Select `casl2-debug-1.0.0.vsix`

### From Source (Development)

```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
```

Then press F5 in VS Code to launch the Extension Development Host.

## Quick Start

1. Open a CASL2 file (`.cas`)
2. Press `F5` to start debugging
3. Set breakpoints by clicking in the gutter
4. Use debug controls:
   - `F5` - Continue
   - `F10` - Step Over
   - `F11` - Step Into
   - `Shift+F11` - Step Out
   - `Shift+F5` - Stop

## Configuration

Create `.vscode/launch.json` in your project:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "casl2",
      "request": "launch",
      "name": "Debug CASL2",
      "program": "${file}",
      "stopOnEntry": true,
      "debugServer": 4711,
      "c2c2Path": "c2c2"
    }
  ]
}
```

### Options

- `program`: Path to CASL2 source file (use `${file}` for current file)
- `stopOnEntry`: Stop at program entry (default: true)
- `debugServer`: TCP port for debug adapter (default: 4711)
- `c2c2Path`: Path to c2c2 executable (default: "c2c2")

## Requirements

- VS Code 1.75.0 or higher
- c2c2 executable with DAP support (`-dap` flag)
- Node.js 18+ and pnpm (for building)

## Project Structure

```
vscode-casl2-debug/
├── src/
│   └── extension.ts              # Extension implementation
├── syntaxes/
│   └── casl2.tmLanguage.json     # Syntax highlighting
├── out/                           # Compiled JavaScript
├── package.json                   # Extension manifest
├── tsconfig.json                  # TypeScript config
└── README.md                      # User documentation
```

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed development instructions.

### Build Commands

```bash
# Install dependencies
pnpm install

# Compile TypeScript
pnpm run compile

# Watch for changes
pnpm run watch

# Package extension
pnpm run package
```

## Features in Detail

### Debug Capabilities

- **Launch**: Automatically starts c2c2 DAP server
- **Breakpoints**: Set/clear breakpoints at source lines
- **Stepping**: Step over, into, and out of instructions
- **Variables**: Inspect all COMET2 registers
- **Stack**: View call stack

### Language Support

- **Syntax Highlighting**: 
  - Instructions (LD, ST, ADDA, etc.)
  - Directives (START, END, DS, DC)
  - Registers (GR0-GR7)
  - Numbers (hex and decimal)
  - Labels and comments

- **Auto-completion**: Matching brackets and quotes
- **Comments**: Line comments with `;`

## Screenshots

### Debugging Session
![Debug session showing breakpoints and variables]

### Syntax Highlighting
![CASL2 code with syntax highlighting]

## Troubleshooting

### c2c2 not found
Ensure c2c2 is in your PATH or set the full path:
```json
{
  "c2c2Path": "/full/path/to/c2c2"
}
```

### Port already in use
Change the debug server port:
```json
{
  "debugServer": 4712
}
```

### Extension doesn't activate
- Check VS Code version (≥1.75.0)
- Verify extension is compiled (`pnpm run compile`)
- Check Output → Extension Host for errors

## License

GPL-3.0

## Contributing

Issues and pull requests welcome at https://github.com/f0reachARR/casljs

## Credits

Part of the CASL2/COMET2 project by Osamu Mizuno.
