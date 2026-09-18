# CASL2 Debug

This extension starts `c2c2` in Debug Adapter Protocol mode and connects VS Code
to it over localhost TCP.

Build `c2c2`, install this directory as a VS Code extension, and add:

```json
{
  "type": "casl2",
  "request": "launch",
  "name": "Debug CASL2",
  "program": "${file}",
  "c2c2Path": "/absolute/path/to/c2c2"
}
```

CASL2 `IN` and `OUT` continue to use the `c2c2` process standard input and
standard output. The `input` launch property can supply values to `IN` in order.
