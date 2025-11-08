# Debug Adapter Protocol (DAP) Examples

## VS Code Integration

To debug CASL2 programs in Visual Studio Code:

1. Start the DAP server:
```bash
./c2c2 -dap 4711
```

2. Install a DAP client extension (or create a custom one)

3. Use the launch configuration from `vscode-launch.json`

4. Set breakpoints in your CASL2 source files and start debugging

## Manual DAP Client Example

You can also connect to the DAP server manually using any TCP client:

```bash
# Start server
./c2c2 -dap 4711

# Connect using netcat (in another terminal)
nc localhost 4711
```

Then send DAP protocol messages in JSON format with Content-Length header:

```
Content-Length: 88

{"seq":1,"type":"request","command":"initialize","arguments":{"clientID":"manual-test"}}
```

## Supported DAP Features

- **initialize**: Initialize debug adapter
- **launch**: Start debugging a CASL2 program
- **setBreakpoints**: Set breakpoints at specific lines
- **configurationDone**: Signal that configuration is complete
- **threads**: Get list of threads (always returns single thread)
- **stackTrace**: Get current stack trace
- **scopes**: Get variable scopes (Registers)
- **variables**: Get variable values (PC, FR, GR0-GR7, SP)
- **continue**: Continue program execution
- **next**: Step to next instruction (step over)
- **stepIn**: Step into instruction (same as next for COMET2)
- **stepOut**: Step out of current function
- **pause**: Pause execution
- **disconnect**: End debugging session
- **terminate**: Terminate the program

## Notes

- Standard input/output are reserved for IN/OUT instructions
- The DAP server can handle multiple sequential debugging sessions
- Each client connection creates a new debugging session
