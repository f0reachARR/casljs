# c2c2 Rust Implementation

This is a Rust implementation of the CASL II assembler / COMET II emulator.

## Status

**Work in Progress** - Basic structure implemented, core functionality under development.

### Completed
- Project structure setup
- Command-line argument parsing
- File I/O
- Basic scaffolding for assembler and emulator

### In Progress
- CASL II assembler implementation
- COMET II emulator implementation
- Test compatibility with existing test suite

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
