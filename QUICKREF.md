# CASL2 Quick Reference

## VSCode Extension Features

| Feature | Shortcut | Description |
|---------|----------|-------------|
| Auto-completion | `Ctrl+Space` | Show instruction and register suggestions |
| Hover info | Mouse hover | Display instruction descriptions |
| Go to definition | `Ctrl+Click` (Mac: `Cmd+Click`) | Jump to label definition |
| Show outline | `Ctrl+Shift+O` | Show document symbols |
| Problems panel | `Ctrl+Shift+M` | View syntax errors |

## Instruction Reference

### Control Flow Instructions
```casl2
START          ; Start of program (label START [address])
END            ; End of program

JUMP adr,x     ; Unconditional jump
JPL  adr,x     ; Jump if plus (FR = 0, not negative)
JMI  adr,x     ; Jump if minus (FR = 2)
JZE  adr,x     ; Jump if zero (FR = 1)
JNZ  adr,x     ; Jump if not zero (FR ≠ 1)
JOV  adr,x     ; Jump if overflow (FR = 4)

CALL adr,x     ; Call subroutine
RET            ; Return from subroutine
```

### Arithmetic Instructions
```casl2
ADDA r,adr,x   ; Add arithmetic: r = r + mem[adr+x]
ADDA r1,r2     ; Add arithmetic: r1 = r1 + r2
SUBA r,adr,x   ; Subtract arithmetic: r = r - mem[adr+x]
SUBA r1,r2     ; Subtract arithmetic: r1 = r1 - r2

ADDL r,adr,x   ; Add logical (unsigned)
ADDL r1,r2     ; Add logical (unsigned)
SUBL r,adr,x   ; Subtract logical (unsigned)
SUBL r1,r2     ; Subtract logical (unsigned)

MULA r,adr,x   ; Multiply arithmetic (extension)
MULA r1,r2     ; Multiply arithmetic (extension)
DIVA r,adr,x   ; Divide arithmetic (extension)
DIVA r1,r2     ; Divide arithmetic (extension)

MULL r,adr,x   ; Multiply logical (extension)
MULL r1,r2     ; Multiply logical (extension)
DIVL r,adr,x   ; Divide logical (extension)
DIVL r1,r2     ; Divide logical (extension)
```

### Logical Instructions
```casl2
AND  r,adr,x   ; Bitwise AND: r = r & mem[adr+x]
AND  r1,r2     ; Bitwise AND: r1 = r1 & r2
OR   r,adr,x   ; Bitwise OR: r = r | mem[adr+x]
OR   r1,r2     ; Bitwise OR: r1 = r1 | r2
XOR  r,adr,x   ; Bitwise XOR: r = r ^ mem[adr+x]
XOR  r1,r2     ; Bitwise XOR: r1 = r1 ^ r2

SLA  r,adr,x   ; Shift left arithmetic
SRA  r,adr,x   ; Shift right arithmetic
SLL  r,adr,x   ; Shift left logical
SRL  r,adr,x   ; Shift right logical
```

### Comparison Instructions
```casl2
CPA  r,adr,x   ; Compare arithmetic: compare r with mem[adr+x]
CPA  r1,r2     ; Compare arithmetic: compare r1 with r2
CPL  r,adr,x   ; Compare logical: compare r with mem[adr+x]
CPL  r1,r2     ; Compare logical: compare r1 with r2
```

### Load/Store Instructions
```casl2
LD   r,adr,x   ; Load: r = mem[adr+x]
LD   r1,r2     ; Load: r1 = r2
ST   r,adr,x   ; Store: mem[adr+x] = r
LAD  r,adr,x   ; Load address: r = adr + x
```

### Stack Instructions
```casl2
PUSH adr,x     ; Push mem[adr+x] to stack
POP  r         ; Pop from stack to r

RPUSH          ; Push all registers (GR1-GR7) to stack
RPOP           ; Pop all registers (GR7-GR1) from stack
```

### I/O Instructions
```casl2
IN   ibuf,ilen ; Read from input into buffer
OUT  obuf,olen ; Write buffer to output
SVC  adr,x     ; Supervisor call
```

### Assembler Directives
```casl2
DS   n         ; Define storage: reserve n words
DC   value     ; Define constant: value (number or string)
DC   'string'  ; Define string constant (null-terminated)
```

### Other
```casl2
NOP            ; No operation
```

## Registers

| Register | Description |
|----------|-------------|
| GR0 | General register 0 (can be used as index) |
| GR1-GR7 | General registers 1-7 |
| PC | Program counter (not directly accessible) |
| SP | Stack pointer (not directly accessible) |
| FR | Flag register (not directly accessible) |

### Flag Register Bits
- Bit 0: Plus (FR_PLUS = 0)
- Bit 1: Zero (FR_ZERO = 1)
- Bit 2: Minus (FR_MINUS = 2)
- Bit 4: Overflow (FR_OVER = 4)

## Number Formats

```casl2
100        ; Decimal number
-50        ; Negative decimal number
#FFFF      ; Hexadecimal number (prefix with #)
#0020      ; Hexadecimal 0x20 (space character)
```

## String Literals

```casl2
DC  'Hello'     ; String constant (null-terminated)
DC  'It''s'     ; Escaped quote (results in "It's")
```

## Label Rules

- Must start with: letter (a-z, A-Z), $, _, %, or .
- Can contain: letters, digits (0-9), $, _, %, or .
- No length limit
- Case-sensitive
- Must be at the start of the line (no leading whitespace)

Examples:
```casl2
START          ; Valid label
LOOP1          ; Valid label
_temp          ; Valid label
$variable      ; Valid label
.local         ; Valid label
```

## Addressing Modes

```casl2
LD  gr1,VAR        ; Direct addressing: adr
LD  gr1,VAR,gr2    ; Indexed addressing: adr+x
LD  gr1,=100       ; Literal: value stored in memory
LD  gr1,=#FF       ; Literal hexadecimal
```

## Example Program Structure

```casl2
MAIN    START           ; Program starts here
        LD    gr0,=0    ; Initialize gr0 to 0
        ADDA  gr0,=10   ; Add 10 to gr0
        ST    gr0,RESULT ; Store result
        CALL  PRINT     ; Call subroutine
        RET             ; Return to system

PRINT                   ; Subroutine starts here
        RPUSH           ; Save registers
        ; ... print code ...
        RPOP            ; Restore registers
        RET             ; Return from subroutine

RESULT  DS    1         ; Reserve 1 word
        END             ; End of program
```

## Common Patterns

### Loop
```casl2
        LD    gr1,=0    ; Counter
LOOP    CPA   gr1,=10   ; Compare with 10
        JZE   END_LOOP  ; Exit if equal
        ; ... loop body ...
        ADDA  gr1,=1    ; Increment counter
        JUMP  LOOP      ; Continue loop
END_LOOP
```

### If-Then-Else
```casl2
        CPA   gr0,=0    ; Compare gr0 with 0
        JMI   ELSE      ; Jump if negative
THEN                    ; If positive or zero
        ; ... then code ...
        JUMP  ENDIF
ELSE                    ; If negative
        ; ... else code ...
ENDIF
```

### Subroutine with Parameters
```casl2
; Call with parameter in gr1
        LD    gr1,=5
        CALL  FUNC
        ; Result in gr0

FUNC    RPUSH           ; Save registers
        ; Use gr1 as parameter
        ; Return value in gr0
        RPOP            ; Restore registers
        RET
```

## Tips for VSCode Extension

1. **Quick Navigation**: Use `Ctrl+Shift+O` to see all labels
2. **Find References**: Right-click on a label → "Find All References"
3. **Rename**: Right-click on a label → "Rename Symbol"
4. **Format**: Keep consistent indentation (tabs or spaces)
5. **Comments**: Use `;` for line comments
6. **Save Often**: Diagnostics update on save

## Building and Running

```bash
# Build with c2c2
go build -o c2c2 .

# Assemble only
./c2c2 -c program.cas

# Assemble and run
./c2c2 -r program.cas

# Quiet mode with inputs
./c2c2 -Q program.cas 10 20 30
```

## Resources

- Full documentation: README.md
- LSP guide: LSP_GUIDE.md
- Grammar specification: GRAMMAR.md
- Extension README: vscode-extension/README.md
