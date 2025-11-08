# DAP Quick Reference

## Starting the DAP Server

```bash
./c2c2 -dap 4711
```

The server will listen on `127.0.0.1:4711` and output:
```
DAP server listening on port 4711
```

## DAP Protocol Flow

### 1. Initialize Session

Client → Server:
```json
{
  "seq": 1,
  "type": "request",
  "command": "initialize",
  "arguments": {
    "clientID": "vscode",
    "adapterID": "casl2"
  }
}
```

Server → Client (response):
```json
{
  "seq": 1,
  "type": "response",
  "request_seq": 1,
  "success": true,
  "command": "initialize",
  "body": {
    "supportsConfigurationDoneRequest": true,
    "supportsTerminateRequest": true
  }
}
```

Server → Client (event):
```json
{
  "seq": 2,
  "type": "event",
  "event": "initialized"
}
```

### 2. Launch Program

```json
{
  "seq": 2,
  "type": "request",
  "command": "launch",
  "arguments": {
    "program": "/path/to/program.cas",
    "stopOnEntry": true
  }
}
```

### 3. Set Breakpoints

```json
{
  "seq": 3,
  "type": "request",
  "command": "setBreakpoints",
  "arguments": {
    "source": {
      "path": "/path/to/program.cas"
    },
    "breakpoints": [
      { "line": 5 },
      { "line": 10 }
    ]
  }
}
```

### 4. Configuration Done

```json
{
  "seq": 4,
  "type": "request",
  "command": "configurationDone"
}
```

Server will send `stopped` event if `stopOnEntry` was true.

### 5. Get Threads

```json
{
  "seq": 5,
  "type": "request",
  "command": "threads"
}
```

Response:
```json
{
  "body": {
    "threads": [
      { "id": 1, "name": "COMET2" }
    ]
  }
}
```

### 6. Get Stack Trace

```json
{
  "seq": 6,
  "type": "request",
  "command": "stackTrace",
  "arguments": {
    "threadId": 1
  }
}
```

### 7. Get Scopes

```json
{
  "seq": 7,
  "type": "request",
  "command": "scopes",
  "arguments": {
    "frameId": 1
  }
}
```

Response includes "Registers" scope.

### 8. Get Variables

```json
{
  "seq": 8,
  "type": "request",
  "command": "variables",
  "arguments": {
    "variablesReference": 1
  }
}
```

Returns PC, FR, GR0-GR7, SP values.

### 9. Step Execution

Next (step over):
```json
{
  "seq": 9,
  "type": "request",
  "command": "next",
  "arguments": {
    "threadId": 1
  }
}
```

Step In:
```json
{
  "seq": 10,
  "type": "request",
  "command": "stepIn",
  "arguments": {
    "threadId": 1
  }
}
```

Step Out:
```json
{
  "seq": 11,
  "type": "request",
  "command": "stepOut",
  "arguments": {
    "threadId": 1
  }
}
```

### 10. Continue

```json
{
  "seq": 12,
  "type": "request",
  "command": "continue",
  "arguments": {
    "threadId": 1
  }
}
```

### 11. Pause

```json
{
  "seq": 13,
  "type": "request",
  "command": "pause",
  "arguments": {
    "threadId": 1
  }
}
```

### 12. Disconnect

```json
{
  "seq": 14,
  "type": "request",
  "command": "disconnect"
}
```

## Events from Server

### Stopped Event

Sent when execution stops (breakpoint, step, pause, etc.):

```json
{
  "type": "event",
  "event": "stopped",
  "body": {
    "reason": "breakpoint",
    "threadId": 1,
    "allThreadsStopped": true
  }
}
```

Reasons:
- `"entry"` - stopped at entry point
- `"step"` - stopped after step
- `"breakpoint"` - stopped at breakpoint
- `"pause"` - stopped by pause request
- `"exception"` - stopped due to error

### Terminated Event

Sent when program ends:

```json
{
  "type": "event",
  "event": "terminated"
}
```

## Message Format

All messages use this format:

```
Content-Length: <length>\r\n
\r\n
<JSON content>
```

Example:
```
Content-Length: 88\r\n
\r\n
{"seq":1,"type":"request","command":"initialize","arguments":{"clientID":"test"}}
```

## Register Values Format

Variables are returned with hex and decimal values:

```json
{
  "name": "GR0",
  "value": "#002A (42)",
  "variablesReference": 0
}
```

Where:
- `#002A` is hex value
- `42` is signed decimal value
- `variablesReference: 0` means no child variables

## Testing with netcat

```bash
# Start server
./c2c2 -dap 4711 &

# Connect
nc localhost 4711

# Send initialize (type or paste):
Content-Length: 88

{"seq":1,"type":"request","command":"initialize","arguments":{"clientID":"manual-test"}}

# Press Enter twice after the JSON
```

## Notes

- All communication is over TCP
- Port number is user-specified
- Each connection creates a new debugging session
- Standard input/output remain available for IN/OUT instructions
- Breakpoints are mapped from source lines to memory addresses
