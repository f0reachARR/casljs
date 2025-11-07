#!/bin/bash
# Wrapper script to run the Go c2c2 binary with the same interface as c2c2.js
# This allows the Go version to be tested with the existing Python test suite

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
C2C2_BIN="$SCRIPT_DIR/c2c2"

# Execute the Go binary with all arguments passed through
exec "$C2C2_BIN" "$@"
