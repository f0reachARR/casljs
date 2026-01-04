# CASL2 Language Server Protocol (LSP) Guide

## Overview

This document provides a comprehensive guide to using the CASL2 Language Server Protocol (LSP) implementation and VSCode extension.

## Architecture

### Components

1. **LSP Server** (`cmd/casl2-lsp`): A standalone Go program that implements the Language Server Protocol for CASL2
2. **VSCode Extension** (`vscode-extension`): A Visual Studio Code extension that connects to the LSP server
3. **LSP Library** (`lsp`): Core LSP implementation as a Go package

### How It Works

The LSP server communicates with VSCode using JSON-RPC over stdin/stdout. When you open a CASL2 file in VSCode:

1. The extension activates and launches the LSP server
2. The server analyzes your CASL2 code in real-time
3. Features like auto-completion, hover info, and diagnostics are provided through the LSP protocol

## Features

### 1. Syntax Highlighting

Rich syntax highlighting for CASL2 assembly language with support for:
- Instructions (color-coded by type: control flow, arithmetic, logical, etc.)
- Labels
- Registers (GR0-GR7)
- Numbers (decimal and hexadecimal)
- Strings
- Comments

### 2. Auto-Completion (IntelliSense)

Press `Ctrl+Space` to trigger auto-completion:
- All CASL2 instructions (NOP, LD, ST, ADDA, JUMP, etc.)
- Register names (GR0-GR7)
- Context-aware suggestions based on cursor position

### 3. Hover Information

Hover over any instruction or register to see:
- Instruction description
- Syntax information
- Usage examples

Example:
```casl2
    LD  gr0,VAR  ; Hover over "LD" to see: "Load - LD r,adr,x or LD r1,r2"
```

### 4. Go to Definition

Click on a label while holding `Ctrl` (or `Cmd` on Mac) to jump to its definition:
```casl2
START
    CALL SUBROUTINE  ; Ctrl+Click on SUBROUTINE jumps to its definition
    RET
    
SUBROUTINE          ; Definition location
    RET
    END
```

### 5. Document Symbols (Outline View)

View all labels in the current file:
1. Open the Outline view (Explorer sidebar)
2. See all labels organized hierarchically
3. Click any label to jump to its location

### 6. Real-time Diagnostics

Get immediate feedback on syntax errors:
- Unknown instructions are highlighted
- Error messages appear in the Problems panel
- Hover over errors to see details

Example of detected error:
```casl2
    INVALID gr0,0  ; Error: Unknown instruction: INVALID
```

## Installation

### Prerequisites

- Go 1.24 or later
- Node.js 18 or later (for building the extension)
- Visual Studio Code 1.75 or later

### Building the LSP Server

```bash
# Clone the repository
git clone https://github.com/f0reachARR/casljs
cd casljs

# Build the LSP server
go build -o casl2-lsp ./cmd/casl2-lsp

# Move to a directory in your PATH (optional)
sudo mv casl2-lsp /usr/local/bin/
```

### Installing the VSCode Extension

#### Option 1: Development Mode

```bash
cd vscode-extension
npm install
npm run compile

# Open VSCode
code .

# Press F5 to launch Extension Development Host
```

#### Option 2: Package and Install

```bash
cd vscode-extension
npm install
npm run compile

# Package the extension
npx vsce package

# Install the .vsix file
code --install-extension casl2-language-support-1.0.0.vsix
```

## Configuration

### VSCode Settings

Configure the extension through VSCode settings:

```json
{
  // Enable/disable the language server
  "casl2.languageServer.enabled": true,
  
  // Path to the LSP server executable
  "casl2.languageServer.path": "casl2-lsp"
}
```

### Custom Server Path

If the LSP server is not in your PATH:

```json
{
  "casl2.languageServer.path": "/path/to/casl2-lsp"
}
```

On Windows:
```json
{
  "casl2.languageServer.path": "C:\\path\\to\\casl2-lsp.exe"
}
```

## Usage Examples

### Example 1: Basic Program with Auto-Completion

```casl2
START
    LD      gr0,0       ; Type "LD" and see auto-completion
    ADDA    gr0,=5      ; Type "AD" to filter to ADDA, ADDL
    ST      gr0,RESULT
    RET
    
RESULT  DS  1
    END
```

### Example 2: Label Navigation

```casl2
MAIN    START
        CALL    INIT    ; Ctrl+Click to jump to INIT
        CALL    PROCESS ; Ctrl+Click to jump to PROCESS
        RET

INIT                    ; Definition of INIT
        LD      gr0,0
        RET

PROCESS                 ; Definition of PROCESS
        LD      gr1,10
        RET
        END
```

### Example 3: Error Detection

```casl2
START
    LD      gr0,0
    BADCMD  gr0,1       ; Error: Unknown instruction: BADCMD
    ADDA    gr0,=10
    RET
    END
```

The error will appear:
- In the editor with a red squiggly underline
- In the Problems panel (View → Problems)

## Supported Instructions

### Control Flow
- START, END
- JUMP, JPL, JMI, JZE, JNZ, JOV
- CALL, RET

### Arithmetic Operations
- ADDA, SUBA (arithmetic)
- ADDL, SUBL (logical)
- MULA, DIVA (arithmetic multiply/divide)
- MULL, DIVL (logical multiply/divide)

### Logical Operations
- AND, OR, XOR
- SLA, SRA (arithmetic shift)
- SLL, SRL (logical shift)

### Comparison
- CPA (compare arithmetic)
- CPL (compare logical)

### Memory Operations
- LD (load)
- ST (store)
- LAD (load address)

### Stack Operations
- PUSH, POP
- RPUSH, RPOP (register push/pop)

### I/O Operations
- IN, OUT

### Directives
- DS (define storage)
- DC (define constant)

### Other
- NOP (no operation)
- SVC (supervisor call)

## Troubleshooting

### LSP Server Not Starting

1. Check that the server is built:
   ```bash
   which casl2-lsp  # On Unix-like systems
   where casl2-lsp  # On Windows
   ```

2. Check VSCode output:
   - Open Output panel (View → Output)
   - Select "CASL2 Language Server" from the dropdown

3. Verify configuration:
   - Open Settings (File → Preferences → Settings)
   - Search for "casl2"
   - Check the server path

### Extension Not Activating

1. Check file extension:
   - Files must have `.cas`, `.casl`, or `.casl2` extension

2. Reload VSCode:
   - Command Palette (Ctrl+Shift+P / Cmd+Shift+P)
   - Run "Developer: Reload Window"

3. Check extension logs:
   - Help → Toggle Developer Tools
   - Check Console for errors

### Auto-Completion Not Working

1. Ensure LSP server is running
2. Try manually triggering: Press `Ctrl+Space`
3. Check that the file is recognized as CASL2 (bottom-right corner of VSCode)

### Performance Issues

If the extension is slow:
1. Disable if not needed:
   ```json
   {
     "casl2.languageServer.enabled": false
   }
   ```

2. Check for very large files (the server analyzes the entire file on each change)

## Development

### Running Tests

LSP server tests:
```bash
cd casljs
go test -v ./lsp/...
```

All tests:
```bash
go test -v ./...
```

### Debugging the LSP Server

1. Build with debug symbols:
   ```bash
   go build -gcflags="all=-N -l" -o casl2-lsp ./cmd/casl2-lsp
   ```

2. Attach a debugger (using delve):
   ```bash
   dlv exec ./casl2-lsp
   ```

### Debugging the Extension

1. Open `vscode-extension` in VSCode
2. Set breakpoints in `src/extension.ts`
3. Press F5 to start debugging
4. Extension runs in a new VSCode window

### Adding New Features

To add a new LSP feature:

1. Implement the feature in `lsp/server.go`
2. Add tests in `lsp/server_test.go`
3. Update capabilities in `handleInitialize()`
4. Add corresponding handler method
5. Test with the VSCode extension

Example: Adding document formatting
```go
// In handleInitialize()
capabilities := map[string]interface{}{
    // ... existing capabilities ...
    "documentFormattingProvider": true,
}

// Add handler
func (s *Server) handleDocumentFormatting(id interface{}, params map[string]interface{}) map[string]interface{} {
    // Implementation
}
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Run all tests: `go test -v ./...`
5. Submit a pull request

## License

GPL v3 - See the COPYING file for details.

## Support

For issues and questions:
- GitHub Issues: https://github.com/f0reachARR/casljs/issues
- Repository: https://github.com/f0reachARR/casljs

## References

- [Language Server Protocol Specification](https://microsoft.github.io/language-server-protocol/)
- [VSCode Extension API](https://code.visualstudio.com/api)
- [CASL2 Specification](https://www.ipa.go.jp/shiken/syllabus/ps6vr70000011n4b-att/sample_casl2.pdf)
