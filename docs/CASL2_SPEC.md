# CASL2 Language Specification

## 1. Overview

CASL2 is an assembly language for the COMET2 virtual machine. This document defines the formal grammar and semantics of CASL2.

## 2. Lexical Structure

### 2.1 Character Set
- ASCII characters
- Case-sensitive for labels
- Instructions are uppercase

### 2.2 Tokens

```
Token ::= Label | Instruction | Register | Immediate | String | Number | Separator

Label ::= LabelChar (LabelChar | Digit)*
LabelChar ::= 'a'..'z' | 'A'..'Z' | '$' | '%' | '_' | '.'
Digit ::= '0'..'9'

Instruction ::= 'NOP' | 'LD' | 'ST' | 'LAD' | 'ADDA' | 'SUBA' | 'ADDL' | 'SUBL'
              | 'MULA' | 'DIVA' | 'MULL' | 'DIVL' | 'AND' | 'OR' | 'XOR'
              | 'CPA' | 'CPL' | 'SLA' | 'SRA' | 'SLL' | 'SRL'
              | 'JMI' | 'JNZ' | 'JZE' | 'JUMP' | 'JPL' | 'JOV'
              | 'PUSH' | 'POP' | 'CALL' | 'RET' | 'SVC'
              | 'START' | 'END' | 'DS' | 'DC'
              | 'IN' | 'OUT' | 'RPUSH' | 'RPOP'

Register ::= 'GR' Digit | Digit
Immediate ::= '#' HexDigit HexDigit HexDigit HexDigit
HexDigit ::= Digit | 'A'..'F' | 'a'..'f'

Number ::= ['+' | '-'] Digit+
String ::= '\'' StringChar* '\''
StringChar ::= <any char except '\'> | '\'\''

Separator ::= ',' | Whitespace
Whitespace ::= ' ' | '\t'
Comment ::= ';' <any char>* EOL
```

## 3. Syntax (BNF Grammar)

### 3.1 Program Structure

```bnf
<program>        ::= <line>*

<line>           ::= <empty-line>
                   | <comment-line>
                   | <instruction-line>
                   | <label-only-line>

<empty-line>     ::= <whitespace>* <eol>
<comment-line>   ::= <whitespace>* <comment> <eol>
<label-only-line>::= <label> <whitespace>* <eol>

<instruction-line> ::= [<label>] <whitespace>+ <instruction> [<whitespace>+ <operands>] [<comment>] <eol>
```

### 3.2 Instructions

```bnf
<instruction>    ::= <op-no-operand>
                   | <op-one-reg>
                   | <op-reg-adr>
                   | <op-adr>
                   | <op-two-reg>
                   | <pseudo-inst>
                   | <macro-inst>

<op-no-operand>  ::= 'NOP' | 'RET'

<op-one-reg>     ::= 'POP' <whitespace>+ <register>

<op-reg-adr>     ::= ('LD' | 'ST' | 'LAD' | 'ADDA' | 'SUBA' | 'ADDL' | 'SUBL'
                   | 'MULA' | 'DIVA' | 'MULL' | 'DIVL' | 'AND' | 'OR' | 'XOR'
                   | 'CPA' | 'CPL' | 'SLA' | 'SRA' | 'SLL' | 'SRL')
                   <whitespace>+ <register> ',' <address> [',' <register>]

<op-adr>         ::= ('JMI' | 'JNZ' | 'JZE' | 'JUMP' | 'JPL' | 'JOV' | 'PUSH' | 'CALL' | 'SVC')
                   <whitespace>+ <address> [',' <register>]

<op-two-reg>     ::= ('LD' | 'ADDA' | 'SUBA' | 'ADDL' | 'SUBL' | 'MULA' | 'DIVA'
                   | 'MULL' | 'DIVL' | 'AND' | 'OR' | 'XOR' | 'CPA' | 'CPL')
                   <whitespace>+ <register> ',' <register>

<pseudo-inst>    ::= <start-inst> | <end-inst> | <ds-inst> | <dc-inst>

<start-inst>     ::= 'START' [<whitespace>+ <label>]
<end-inst>       ::= 'END'
<ds-inst>        ::= 'DS' <whitespace>+ <number>
<dc-inst>        ::= 'DC' <whitespace>+ <dc-value> (',' <dc-value>)*
<dc-value>       ::= <number> | <string> | <label> | <immediate>

<macro-inst>     ::= <in-inst> | <out-inst> | <rpush-inst> | <rpop-inst>

<in-inst>        ::= 'IN' <whitespace>+ <label> ',' <label>
<out-inst>       ::= 'OUT' <whitespace>+ <label> ',' <label>
<rpush-inst>     ::= 'RPUSH'
<rpop-inst>      ::= 'RPOP'
```

### 3.3 Operands

```bnf
<operands>       ::= <operand> (',' <operand>)*

<operand>        ::= <register> | <address> | <number> | <string>

<register>       ::= 'GR' <digit> | <digit>
<digit>          ::= '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7'

<address>        ::= <label> | <number> | <immediate> | <literal>

<literal>        ::= '=' <literal-value>
<literal-value>  ::= <number> | <string> | <immediate>

<label>          ::= <label-char> (<label-char> | <digit>)*
<label-char>     ::= 'a'..'z' | 'A'..'Z' | '$' | '%' | '_' | '.'

<number>         ::= ['+' | '-'] <digit>+

<immediate>      ::= '#' <hex-digit> <hex-digit> <hex-digit> <hex-digit>
<hex-digit>      ::= '0'..'9' | 'A'..'F' | 'a'..'f'

<string>         ::= '\'' <string-char>* '\''
<string-char>    ::= <any-char-except-quote> | '\'\''
```

## 4. Semantics

### 4.1 Program Structure

- A program consists of one or more sections
- Each section starts with `START` and ends with `END`
- Labels have scope within their section

### 4.2 Label Scoping Rules

```
Scope ::= SectionLabel ':' LocalLabel

- Within a section, labels are prefixed with the section name
- START instruction creates a new scope
- CALL instruction can reference labels from other scopes
- Literal labels are unique and prefixed with '=' and a counter
```

### 4.3 Instruction Types

#### 4.3.1 Arithmetic/Logical Operations (Op5)
- Format: `INST GR, adr[, XR]` or `INST GR, GR`
- Instructions: LD, ADDA, SUBA, ADDL, SUBL, MULA, DIVA, MULL, DIVL, AND, OR, XOR, CPA, CPL
- Size: 1 word (reg-reg) or 2 words (reg-mem)

#### 4.3.2 Memory Operations (Op1)
- Format: `INST GR, adr[, XR]`
- Instructions: ST, LAD, SLA, SRA, SLL, SRL
- Size: 2 words

#### 4.3.3 Branch Operations (Op2)
- Format: `INST adr[, XR]`
- Instructions: JMI, JNZ, JZE, JUMP, JPL, JOV, PUSH, CALL, SVC
- Size: 2 words

#### 4.3.4 Register Operations (Op3)
- Format: `INST GR`
- Instructions: POP
- Size: 1 word

#### 4.3.5 No-operand Operations (Op4)
- Format: `INST`
- Instructions: NOP, RET
- Size: 1 word

### 4.4 Pseudo Instructions

#### START
- Syntax: `label START [entry_label]`
- Creates a new program section with scope `label`
- Optional entry point within the section

#### END
- Syntax: `END`
- Ends current program section
- Expands accumulated literals

#### DS (Define Storage)
- Syntax: `DS count`
- Allocates `count` words of storage initialized to 0

#### DC (Define Constant)
- Syntax: `DC value1, value2, ...`
- Defines constants (numbers, strings, labels)
- Strings are null-terminated

### 4.5 Macros

#### IN
- Syntax: `IN buffer_label, length_label`
- Expands to: PUSH, PUSH, LAD, LAD, SVC, POP, POP
- Reads input into buffer

#### OUT
- Syntax: `OUT buffer_label, length_label`
- Expands to: PUSH, PUSH, LAD, LAD, SVC, POP, POP
- Outputs buffer content

#### RPUSH
- Syntax: `RPUSH`
- Pushes GR1-GR7 onto stack

#### RPOP
- Syntax: `RPOP`
- Pops GR7-GR1 from stack

## 5. Machine Code Format

### 5.1 Instruction Format

```
Word 1: [opcode (8 bits)] [r1 (4 bits)] [r2/x (4 bits)]
Word 2: [address (16 bits)] (if applicable)
```

### 5.2 Opcode Table

| Instruction | Opcode | Format |
|-------------|--------|--------|
| NOP         | 0x00   | op     |
| LD (r,a,x)  | 0x10   | op r a x |
| LD (r,r)    | 0x14   | op r r |
| ST          | 0x11   | op r a x |
| LAD         | 0x12   | op r a x |
| ADDA (r,a,x)| 0x20   | op r a x |
| ADDA (r,r)  | 0x24   | op r r |
| ... (see instruction.rs for complete table) |

## 6. Error Handling

### Syntax Errors
- Invalid instruction name
- Invalid register (must be 0-7)
- Invalid label format
- Operand count mismatch

### Semantic Errors
- Undefined label
- Duplicate label definition
- Label scope violation
- GR0 used as index register
- Stack overflow/underflow

### Runtime Errors
- Division by zero (sets OF and ZF)
- Stack overflow
- Stack underflow
- Illegal instruction
