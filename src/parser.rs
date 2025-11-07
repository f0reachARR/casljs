// Recursive Descent Parser for CASL2

use crate::ast::*;
use crate::lexer::Token;

pub struct Parser {
    tokens: Vec<(Token, usize)>,
    pos: usize,
}

impl Parser {
    pub fn new(tokens: Vec<(Token, usize)>) -> Self {
        Parser { tokens, pos: 0 }
    }

    fn peek(&self) -> Option<&Token> {
        if self.pos < self.tokens.len() {
            Some(&self.tokens[self.pos].0)
        } else {
            None
        }
    }

    fn advance(&mut self) -> Option<Token> {
        if self.pos < self.tokens.len() {
            let token = self.tokens[self.pos].0.clone();
            self.pos += 1;
            Some(token)
        } else {
            None
        }
    }

    fn current_line(&self) -> usize {
        if self.pos > 0 && self.pos <= self.tokens.len() {
            self.tokens[self.pos - 1].1
        } else if self.pos < self.tokens.len() {
            self.tokens[self.pos].1
        } else {
            0
        }
    }

    fn expect(&mut self, expected: &str) -> Result<(), String> {
        match self.peek() {
            Some(token) if format!("{:?}", token).contains(expected) => {
                self.advance();
                Ok(())
            }
            Some(token) => Err(format!(
                "Line {}: Expected {}, got {:?}",
                self.current_line(),
                expected,
                token
            )),
            None => Err(format!(
                "Line {}: Expected {}, got EOF",
                self.current_line(),
                expected
            )),
        }
    }

    fn skip_newlines(&mut self) {
        while matches!(self.peek(), Some(Token::Newline) | Some(Token::Comment(_))) {
            self.advance();
        }
    }

    pub fn parse(&mut self) -> Result<Program, String> {
        let mut lines = Vec::new();

        self.skip_newlines();

        while !matches!(self.peek(), Some(Token::Eof) | None) {
            let line = self.parse_line()?;
            if line.label.is_some() || line.instruction.is_some() {
                lines.push(line);
            }
            self.skip_newlines();
        }

        Ok(Program { lines })
    }

    fn parse_line(&mut self) -> Result<Line, String> {
        let line_number = self.current_line();
        let mut label = None;
        let mut instruction = None;

        // Skip comments at the start of line
        while matches!(self.peek(), Some(Token::Comment(_))) {
            self.advance();
        }

        // Check for label
        if let Some(Token::Label(l)) = self.peek() {
            label = Some(l.clone());
            self.advance();

            // After label, there might be an instruction or just a newline
            if matches!(self.peek(), Some(Token::Newline) | Some(Token::Comment(_)) | None) {
                // Label only line
                return Ok(Line {
                    label,
                    instruction: None,
                    line_number,
                });
            }
        }

        // Parse instruction if present
        if let Some(Token::Instruction(inst_name)) = self.peek() {
            let inst_name = inst_name.clone();
            self.advance();
            instruction = Some(self.parse_instruction(&inst_name)?);
        }

        // Skip to end of line
        while matches!(self.peek(), Some(Token::Comment(_))) {
            self.advance();
        }

        if matches!(self.peek(), Some(Token::Newline)) {
            self.advance();
        }

        Ok(Line {
            label,
            instruction,
            line_number,
        })
    }

    fn parse_instruction(&mut self, name: &str) -> Result<Instruction, String> {
        match name {
            // No operand instructions
            "NOP" => Ok(Instruction::NoOperand(NoOperandInst::Nop)),
            "RET" => Ok(Instruction::NoOperand(NoOperandInst::Ret)),

            // POP (one register)
            "POP" => {
                let reg = self.parse_register()?;
                Ok(Instruction::OneReg(OneRegInst::Pop, reg))
            }

            // Pseudo instructions
            "START" => {
                let entry_label = if matches!(self.peek(), Some(Token::Label(_))) {
                    if let Some(Token::Label(l)) = self.advance() {
                        Some(l)
                    } else {
                        None
                    }
                } else {
                    None
                };
                Ok(Instruction::Start(entry_label))
            }

            "END" => Ok(Instruction::End),

            "DS" => {
                let count = self.parse_number()?;
                Ok(Instruction::Ds(count))
            }

            "DC" => {
                let values = self.parse_dc_values()?;
                Ok(Instruction::Dc(values))
            }

            // Macros
            "IN" => {
                let buf = self.parse_label()?;
                self.expect_comma()?;
                let len = self.parse_label()?;
                Ok(Instruction::In(buf, len))
            }

            "OUT" => {
                let buf = self.parse_label()?;
                self.expect_comma()?;
                let len = self.parse_label()?;
                Ok(Instruction::Out(buf, len))
            }

            "RPUSH" => Ok(Instruction::Rpush),
            "RPOP" => Ok(Instruction::Rpop),

            // PUSH and CALL (address instructions)
            _ if AddrInst::from_str(name).is_some() => {
                let addr_inst = AddrInst::from_str(name).unwrap();
                let addr = self.parse_address()?;
                let index = if matches!(self.peek(), Some(Token::Comma)) {
                    self.advance();
                    Some(self.parse_register()?)
                } else {
                    None
                };
                Ok(Instruction::Addr(addr_inst, addr, index))
            }

            // Register-Address instructions
            _ if RegAddrInst::from_str(name).is_some() => {
                let reg_addr_inst = RegAddrInst::from_str(name).unwrap();
                let reg1 = self.parse_register()?;
                self.expect_comma()?;

                // Check if second operand is a register (for two-reg format)
                if matches!(self.peek(), Some(Token::Register(_))) && reg_addr_inst.can_be_two_reg() {
                    let reg2 = self.parse_register()?;
                    let two_reg_inst = TwoRegInst::from_reg_addr(reg_addr_inst).unwrap();
                    Ok(Instruction::TwoReg(two_reg_inst, reg1, reg2))
                } else {
                    // Register-Address format
                    let addr = self.parse_address()?;
                    let index = if matches!(self.peek(), Some(Token::Comma)) {
                        self.advance();
                        Some(self.parse_register()?)
                    } else {
                        None
                    };
                    Ok(Instruction::RegAddr(reg_addr_inst, reg1, addr, index))
                }
            }

            _ => Err(format!("Unknown instruction: {}", name)),
        }
    }

    fn parse_register(&mut self) -> Result<Register, String> {
        match self.advance() {
            Some(Token::Register(n)) => Register::from_u8(n)
                .ok_or_else(|| format!("Invalid register number: {}", n)),
            Some(token) => Err(format!(
                "Line {}: Expected register, got {:?}",
                self.current_line(),
                token
            )),
            None => Err(format!(
                "Line {}: Expected register, got EOF",
                self.current_line()
            )),
        }
    }

    fn parse_address(&mut self) -> Result<Address, String> {
        match self.peek() {
            Some(Token::Label(l)) => {
                let label = l.clone();
                self.advance();
                Ok(Address::Label(label))
            }
            Some(Token::Number(_)) => {
                let num = self.parse_number()?;
                Ok(Address::Number(num))
            }
            Some(Token::Immediate(_)) => {
                if let Some(Token::Immediate(imm)) = self.advance() {
                    Ok(Address::Immediate(imm))
                } else {
                    unreachable!()
                }
            }
            Some(Token::Literal(lit)) => {
                let literal = lit.clone();
                self.advance();
                Ok(Address::Literal(literal))
            }
            Some(token) => Err(format!(
                "Line {}: Expected address, got {:?}",
                self.current_line(),
                token
            )),
            None => Err(format!(
                "Line {}: Expected address, got EOF",
                self.current_line()
            )),
        }
    }

    fn parse_label(&mut self) -> Result<String, String> {
        match self.advance() {
            Some(Token::Label(l)) => Ok(l),
            Some(token) => Err(format!(
                "Line {}: Expected label, got {:?}",
                self.current_line(),
                token
            )),
            None => Err(format!(
                "Line {}: Expected label, got EOF",
                self.current_line()
            )),
        }
    }

    fn parse_number(&mut self) -> Result<i32, String> {
        match self.advance() {
            Some(Token::Number(n)) => Ok(n),
            Some(token) => Err(format!(
                "Line {}: Expected number, got {:?}",
                self.current_line(),
                token
            )),
            None => Err(format!(
                "Line {}: Expected number, got EOF",
                self.current_line()
            )),
        }
    }

    fn parse_dc_values(&mut self) -> Result<Vec<DcValue>, String> {
        let mut values = Vec::new();

        loop {
            let value = match self.peek() {
                Some(Token::Number(_)) => {
                    let n = self.parse_number()?;
                    DcValue::Number(n)
                }
                Some(Token::String(_)) => {
                    if let Some(Token::String(s)) = self.advance() {
                        DcValue::String(s)
                    } else {
                        unreachable!()
                    }
                }
                Some(Token::Label(_)) => {
                    let l = self.parse_label()?;
                    DcValue::Label(l)
                }
                Some(Token::Immediate(_)) => {
                    if let Some(Token::Immediate(imm)) = self.advance() {
                        DcValue::Immediate(imm)
                    } else {
                        unreachable!()
                    }
                }
                Some(token) => {
                    return Err(format!(
                        "Line {}: Expected DC value, got {:?}",
                        self.current_line(),
                        token
                    ))
                }
                None => {
                    return Err(format!(
                        "Line {}: Expected DC value, got EOF",
                        self.current_line()
                    ))
                }
            };

            values.push(value);

            // Check for comma (more values)
            if matches!(self.peek(), Some(Token::Comma)) {
                self.advance();
            } else {
                break;
            }
        }

        Ok(values)
    }

    fn expect_comma(&mut self) -> Result<(), String> {
        match self.advance() {
            Some(Token::Comma) => Ok(()),
            Some(token) => Err(format!(
                "Line {}: Expected comma, got {:?}",
                self.current_line(),
                token
            )),
            None => Err(format!(
                "Line {}: Expected comma, got EOF",
                self.current_line()
            )),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::lexer::Lexer;

    #[test]
    fn test_parse_simple() {
        let input = "MAIN START\n  LD GR1, DATA\n  RET\nDATA DC 10\n  END";
        let mut lexer = Lexer::new(input);
        let tokens = lexer.tokenize().unwrap();
        let mut parser = Parser::new(tokens);
        let program = parser.parse().unwrap();

        assert!(program.lines.len() > 0);
    }
}
