# CASL2 Grammar Specification

## Lexical Elements

### Tokens
```
LABEL       ::= [a-zA-Z$%_.][0-9a-zA-Z$%_.]*
INSTRUCTION ::= [A-Z]+
NUMBER      ::= [+-]?[0-9]+
HEXNUM      ::= #[0-9a-fA-F]+
REGISTER    ::= GR[0-7]
STRING      ::= '[^']*'  (with '' as escaped quote)
COMMA       ::= ,
EQUALS      ::= =
WHITESPACE  ::= [ \t]+
NEWLINE     ::= \n
COMMENT     ::= ;.*
```

## Grammar (LL(1))

```
Program     ::= Line*

Line        ::= EmptyLine
              | LabelLine
              | InstructionLine
              | LabelInstructionLine

EmptyLine   ::= WHITESPACE? COMMENT? NEWLINE

LabelLine   ::= LABEL WHITESPACE? COMMENT? NEWLINE

InstructionLine ::= WHITESPACE INSTRUCTION Operands? COMMENT? NEWLINE

LabelInstructionLine ::= LABEL WHITESPACE INSTRUCTION Operands? COMMENT? NEWLINE

Operands    ::= WHITESPACE OperandList

OperandList ::= Operand (COMMA Operand)*

Operand     ::= Register
              | Address
              | Literal
              | Number
              | String

Register    ::= REGISTER

Address     ::= LABEL
              | Number
              | HexNumber

Literal     ::= EQUALS (Number | HexNumber | String)

Number      ::= NUMBER

HexNumber   ::= HEXNUM

String      ::= STRING
```

## FIRST and FOLLOW Sets

```
FIRST(Program) = {LABEL, WHITESPACE, INSTRUCTION, NEWLINE, EOF}
FIRST(Line) = {LABEL, WHITESPACE, INSTRUCTION, NEWLINE}
FIRST(EmptyLine) = {WHITESPACE, NEWLINE}
FIRST(LabelLine) = {LABEL}
FIRST(InstructionLine) = {WHITESPACE}
FIRST(LabelInstructionLine) = {LABEL}
FIRST(Operands) = {WHITESPACE}
FIRST(OperandList) = {REGISTER, LABEL, NUMBER, HEXNUM, EQUALS, STRING}
FIRST(Operand) = {REGISTER, LABEL, NUMBER, HEXNUM, EQUALS, STRING}

FOLLOW(Program) = {EOF}
FOLLOW(Line) = {LABEL, WHITESPACE, INSTRUCTION, NEWLINE, EOF}
FOLLOW(Operands) = {COMMENT, NEWLINE}
FOLLOW(OperandList) = {COMMENT, NEWLINE}
FOLLOW(Operand) = {COMMA, COMMENT, NEWLINE}
```

## Parser Implementation Strategy

1. **Lexer**: Tokenize input into stream of tokens
2. **Parser**: LL(1) recursive descent parser
3. **No Regular Expressions**: Use character-by-character scanning

## Token Types
```go
type TokenType int

const (
    TOKEN_EOF TokenType = iota
    TOKEN_NEWLINE
    TOKEN_LABEL
    TOKEN_INSTRUCTION
    TOKEN_REGISTER
    TOKEN_NUMBER
    TOKEN_HEXNUM
    TOKEN_STRING
    TOKEN_COMMA
    TOKEN_EQUALS
    TOKEN_WHITESPACE
    TOKEN_COMMENT
)

type Token struct {
    Type    TokenType
    Value   string
    Line    int
    Column  int
}
```

## Lexer State Machine

```
State: START
  [a-zA-Z$%_.]  -> STATE_LABEL
  [0-9+-]       -> STATE_NUMBER
  #             -> STATE_HEXNUM
  '             -> STATE_STRING
  GR            -> STATE_REGISTER (check next char)
  ,             -> TOKEN_COMMA
  =             -> TOKEN_EQUALS
  [ \t]         -> TOKEN_WHITESPACE
  ;             -> STATE_COMMENT
  \n            -> TOKEN_NEWLINE
  EOF           -> TOKEN_EOF

State: STATE_LABEL
  [0-9a-zA-Z$%_.]  -> STATE_LABEL
  other            -> Check if instruction name (all uppercase), else TOKEN_LABEL

State: STATE_NUMBER
  [0-9]         -> STATE_NUMBER
  other         -> TOKEN_NUMBER

State: STATE_HEXNUM
  [0-9a-fA-F]   -> STATE_HEXNUM
  other         -> TOKEN_HEXNUM

State: STATE_STRING
  '             -> Check next: if ' then continue, else TOKEN_STRING
  other         -> STATE_STRING

State: STATE_COMMENT
  \n            -> TOKEN_COMMENT
  other         -> STATE_COMMENT

State: STATE_REGISTER (after GR)
  [0-7]         -> TOKEN_REGISTER
  other         -> Back to LABEL
```
