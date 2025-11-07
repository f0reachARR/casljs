# c2c2 Go Implementation

This directory contains a Go implementation of the CASL2 assembler and COMET2 emulator.

## Features

- Full CASL2 assembler with all pseudo-instructions (START, END, DS, DC, IN, OUT, RPUSH, RPOP)
- Complete COMET2 emulator with all instructions
- Interactive debugger with commands: run, step, print, dump, stack, disasm, help, quit
- Command-line compatible with the JavaScript version
- Fast execution (compiled Go binary)
- Comprehensive test suite (28 test cases)

## Building

Requirements:
- Go 1.16 or later

Build the binary:
```bash
go build -o c2c2 .
```

## Usage

Basic usage:
```bash
./c2c2 [options] <casl2file> [input1 ...]
```

Options:
- `-V` - Output version number
- `-a` - Show detailed assembly listing
- `-c` - Assemble only (don't run)
- `-r` - Run immediately after assembly
- `-n` - Disable color output
- `-q` - Quiet mode (suppress banner)
- `-Q` - Very quiet mode (implies -q and -r, suppress all prompts)

### Examples

Assemble and run a program:
```bash
./c2c2 program.cas
```

Assemble and run with inputs:
```bash
./c2c2 -n -Q program.cas 10 20 30
```

Show assembly listing:
```bash
./c2c2 -a -c program.cas
```

Run in interactive debugger:
```bash
./c2c2 program.cas
# Then use commands: run, step, print, help, etc.
```

## Testing

Run all tests:
```bash
go test -v
```

Run a specific test:
```bash
go test -v -run TestC2C2Samples/sample11.cas
```

## Implementation Files

- `main.go` - Main program, CLI parsing, and I/O handling
- `assembler.go` - CASL2 assembler (pass1 and pass2)
- `emulator.go` - COMET2 emulator and instruction execution
- `commands.go` - Interactive debugger commands
- `c2c2_test.go` - Test suite

## Differences from c2c2.js

The Go implementation is functionally identical to the JavaScript version:
- Same command-line options
- Same assembler behavior
- Same emulator behavior
- Same output format
- All 28 test cases produce identical output

The only differences are:
- Written in Go instead of JavaScript
- Faster execution (compiled vs interpreted)
- No dependency on Node.js
- Slightly different error handling internally (but same user-visible behavior)

## Compatibility

This implementation maintains 100% compatibility with the JavaScript version:
- All test cases pass
- Output format is identical
- Command-line interface is the same
- Can be used as a drop-in replacement for c2c2.js
