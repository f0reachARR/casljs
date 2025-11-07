# c2c2 Rust Implementation

This is a Rust implementation of the CASL II assembler / COMET II emulator.

## Status

**Completed** - Full CASL II assembler and COMET II emulator implementation.

### Implemented Features

#### Lexer
- Hand-written tokenizer (no regex)
- Case-insensitive instruction and register parsing
- Support for all CASL2 tokens: labels, instructions, registers, numbers, immediates, strings, literals, comments
- 16 comprehensive unit tests

#### Parser
- Recursive descent parser (LL(1) style)
- Type-safe AST construction
- All instruction types and pseudo-instructions
- 17 comprehensive unit tests

#### Assembler
- Two-pass code generation
- Pass 1: Symbol table with label scoping (START/END blocks)
- Pass 2: Machine code generation with label resolution
- Literal addressing (=value) - literals placed at END
- Macro expansion: IN, OUT, RPUSH, RPOP
- DC string handling (each character → 1 word)
- All CASL2 instructions and addressing modes

#### Emulator
- Full COMET2 virtual machine
- 8 general-purpose registers (GR0-GR7), PC, SP, FR
- 64K word memory
- All arithmetic and logical operations
- Comparison, branch, and jump instructions
- Stack operations and subroutine support
- System calls: IN, OUT, termination, error handling

### Test Results
- All unit tests pass (33/33)
- Successfully assembles sample programs
- Compatible with test suite format

## Building

```bash
cargo build --release
```

## Usage

```bash
./target/release/c2c2 [options] <casl2file> [input1 ...]
```

### Options
- `-V, --version` - Output the version number
- `-a, --all` - [casl2] Show detailed info
- `-c, --casl` - [casl2] Apply casl2 only
- `-r, --run` - [comet2] Run immediately
- `-n, --nocolor` - [casl2/comet2] Disable color messages
- `-q, --quiet` - [casl2/comet2] Be quiet
- `-Q, --QuietRun` - [comet2] Be QUIET! (implies -q and -r)
- `-h, --help` - Display help for command

## Implementation Notes

This Rust implementation is being developed without external crates (regex, clap, anyhow) to work in restricted environments. The assembler and emulator logic is being ported from the original JavaScript implementation (c2c2.js).

## Testing

```bash
cd test
python3 -m pytest c2c2_test.py -v
```

## License

Same as original c2c2:
- CASL assembler / COMET emulator
  - Copyright (C) 1998-2000 Hiroyuki Ohsaki
- CASL II assembler / COMET II emulator
  - Copyright (C) 2001-2023 Osamu Mizuno

This program is free software; you can redistribute it and/or modify it
under the terms of the GNU General Public License.
