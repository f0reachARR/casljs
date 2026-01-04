# CASL2 Language Support for VSCode

CASL2 assembly language support for Visual Studio Code with Language Server Protocol (LSP) integration.

## Features

- **Syntax Highlighting**: Rich syntax highlighting for CASL2 assembly language
- **IntelliSense**: Auto-completion for instructions, registers, and labels
- **Hover Information**: Display instruction descriptions on hover
- **Go to Definition**: Jump to label definitions
- **Document Symbols**: Outline view showing all labels in the document
- **Diagnostics**: Real-time syntax error checking

## Requirements

This extension requires the CASL2 Language Server (`casl2-lsp`) to be installed and available in your system PATH, or you can configure the path to the executable in the settings.

## Installation

### Installing the Language Server

1. Build the language server from the CASL2 repository:
   ```bash
   cd /path/to/casljs
   go build -o casl2-lsp ./cmd/casl2-lsp
   ```

2. Move the executable to a location in your PATH or configure the path in VSCode settings.

### Installing the Extension

#### Option 1: From VSIX (Recommended for local installation)
1. Package the extension:
   ```bash
   cd vscode-extension
   npm install
   npm run compile
   npx vsce package
   ```

2. Install the generated `.vsix` file:
   - Open VSCode
   - Go to Extensions view (Ctrl+Shift+X)
   - Click the "..." menu at the top
   - Select "Install from VSIX..."
   - Choose the generated `.vsix` file

#### Option 2: Development Mode
1. Open the `vscode-extension` folder in VSCode
2. Run `npm install` to install dependencies
3. Press F5 to launch the extension in a new VSCode window

## Extension Settings

This extension contributes the following settings:

* `casl2.languageServer.enabled`: Enable/disable the CASL2 language server (default: `true`)
* `casl2.languageServer.path`: Path to the CASL2 language server executable (default: `casl2-lsp`)

## Usage

1. Open any `.cas`, `.casl`, or `.casl2` file in VSCode
2. The extension will automatically activate and provide language support
3. Start typing to see auto-completion suggestions
4. Hover over instructions to see their descriptions
5. Use Ctrl+Click (Cmd+Click on Mac) on labels to jump to their definitions

## Supported Instructions

The extension supports all CASL2 instructions including:
- Arithmetic: ADDA, SUBA, ADDL, SUBL, MULA, DIVA, MULL, DIVL
- Logical: AND, OR, XOR, SLA, SRA, SLL, SRL
- Comparison: CPA, CPL
- Control Flow: JMI, JNZ, JZE, JUMP, JPL, JOV, CALL, RET
- Memory: LD, ST, LAD, PUSH, POP, RPUSH, RPOP
- I/O: IN, OUT
- Directives: START, END, DS, DC
- Other: NOP, SVC

## Known Issues

None at this time.

## Release Notes

### 1.0.0

Initial release with LSP support:
- Syntax highlighting for CASL2
- Auto-completion for instructions and registers
- Hover information for instructions
- Go to definition for labels
- Document symbols view
- Basic syntax error detection

## Contributing

Contributions are welcome! Please visit the [GitHub repository](https://github.com/f0reachARR/casljs) for more information.

## License

GPL v3 - See the LICENSE file for details.
