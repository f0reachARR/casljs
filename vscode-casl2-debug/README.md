# CASL2 Debugger for VS Code

VS Code extension for debugging CASL2/COMET2 assembly programs.

## Features

- Debug CASL2 programs with full Debug Adapter Protocol support
- Step through code (step, step in, step out)
- Set breakpoints at source lines
- Inspect registers (PC, FR, GR0-GR7, SP)
- View stack traces
- Syntax highlighting for CASL2 assembly language

## Requirements

- c2c2 executable must be in your PATH or specify the path in launch configuration
- VS Code 1.75.0 or higher

## Installation

### From VSIX

1. Download the `.vsix` file
2. In VS Code, go to Extensions view (Ctrl+Shift+X)
3. Click "..." menu → "Install from VSIX..."
4. Select the downloaded `.vsix` file

### From Source

```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
pnpm run package
```

Then install the generated `.vsix` file.

## Usage

### Quick Start

1. Open a CASL2 source file (`.cas`)
2. Press F5 to start debugging
3. The debugger will stop at the entry point
4. Use the debug toolbar to step through your code

### Launch Configuration

Create a `.vscode/launch.json` in your workspace:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "casl2",
      "request": "launch",
      "name": "Debug CASL2 Program",
      "program": "${file}",
      "stopOnEntry": true,
      "debugServer": 4711,
      "c2c2Path": "c2c2"
    }
  ]
}
```

### Configuration Options

- `program` (required): Path to the CASL2 source file
- `stopOnEntry` (optional, default: true): Stop at program entry
- `debugServer` (optional, default: 4711): TCP port for debug adapter
- `c2c2Path` (optional, default: "c2c2"): Path to c2c2 executable

### Setting Breakpoints

Click in the gutter next to line numbers to set breakpoints. Breakpoints are shown as red dots.

### Debugging

- **Continue (F5)**: Resume execution
- **Step Over (F10)**: Execute the current line
- **Step Into (F11)**: Step into function calls
- **Step Out (Shift+F11)**: Step out of current function
- **Pause**: Pause execution
- **Stop (Shift+F5)**: Stop debugging

### Viewing Variables

The Variables pane shows all registers:
- PC: Program Counter
- FR: Flag Register
- GR0-GR7: General Registers
- SP: Stack Pointer

Values are shown in both hexadecimal and decimal format.

## Example Program

```casl2
; Simple addition program
MAIN    START
        LD      GR0, =10        ; Load 10 into GR0
        LD      GR1, =20        ; Load 20 into GR1
        ADDA    GR0, GR1        ; Add GR1 to GR0
        ST      GR0, RESULT     ; Store result
        RET                     ; Return
RESULT  DS      1               ; Result storage
        END
```

## Syntax Highlighting

The extension provides syntax highlighting for:
- Instructions (LD, ST, ADDA, etc.)
- Directives (START, END, DS, DC, etc.)
- Registers (GR0-GR7)
- Labels
- Comments (`;`)
- Numbers (decimal and hexadecimal)
- Strings

## Troubleshooting

### c2c2 not found

Make sure c2c2 is in your PATH, or specify the full path in `c2c2Path`:

```json
{
  "c2c2Path": "/path/to/c2c2"
}
```

### Port already in use

If port 4711 is already in use, specify a different port:

```json
{
  "debugServer": 4712
}
```

### Debug server fails to start

Check that:
1. c2c2 executable has the `-dap` option support
2. The program path is correct and the file exists
3. No firewall is blocking localhost connections

## Development

### Building from Source

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

### Testing

1. Open the extension folder in VS Code
2. Press F5 to launch Extension Development Host
3. Open a CASL2 file and test debugging

## License

GPL-3.0

## Contributing

Issues and pull requests are welcome at https://github.com/f0reachARR/casljs

## Credits

Based on the CASL2/COMET2 implementation by Osamu Mizuno.
