# CASL2/COMET2 Go Implementation Tests

This directory contains tests for the Go implementation of the CASL2 assembler and COMET2 emulator.

## Test Structure

- `samples/`: CASL2 source files used for testing
- `test_expects/`: Expected output files for each test sample
- `c2c2_test.go` (in parent directory): Go test suite

## Running Tests

The tests are written in Go and use the standard Go testing framework.

### Prerequisites

- Go 1.21 or later
- The `c2c2` binary must be built before running tests

### Build and Test

```bash
# From the repository root
go build -o c2c2 .
go test -v
```

### Run tests with coverage

```bash
go test -v -race -coverprofile=coverage.txt -covermode=atomic
```

### Run specific tests

```bash
# Test a specific sample
go test -v -run TestC2C2Samples/sample11.cas
```

## Test Samples

Tests are run against all `sample*.cas` files in the `samples/` directory. Each test:

1. Assembles the CASL2 source file
2. Executes the COMET2 program with predefined inputs
3. Compares the output against the expected output in `test_expects/`

## Expected Output

Expected output for each test is stored in `test_expects/` directory:
- `sampleN.cas.out`: Expected output for `sampleN.cas`

If the actual output differs from the expected output, the test fails and shows a diff.

## Continuous Integration

Tests are automatically run via GitHub Actions on:
- Push to main branch
- Pull requests to main branch
- Multiple OS platforms: Ubuntu, Windows, macOS
- Multiple Go versions: 1.21, 1.22
