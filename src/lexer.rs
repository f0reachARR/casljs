// Lexer (Tokenizer) for CASL2
// Performs lexical analysis without regular expressions

use std::fmt;

#[derive(Debug, Clone, PartialEq)]
pub enum Token {
    // Identifiers and literals
    Label(String),
    Instruction(String),
    Register(u8),          // 0-7
    Number(i32),
    Immediate(u16),        // #XXXX
    String(String),
    Literal(String),       // =value

    // Separators
    Comma,
    Colon,

    // Special
    Comment(String),
    Newline,
    Eof,
}

impl fmt::Display for Token {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match self {
            Token::Label(s) => write!(f, "Label({})", s),
            Token::Instruction(s) => write!(f, "Instruction({})", s),
            Token::Register(r) => write!(f, "GR{}", r),
            Token::Number(n) => write!(f, "Number({})", n),
            Token::Immediate(v) => write!(f, "#{:04X}", v),
            Token::String(s) => write!(f, "String('{}')", s),
            Token::Literal(s) => write!(f, "Literal({})", s),
            Token::Comma => write!(f, ","),
            Token::Colon => write!(f, ":"),
            Token::Comment(s) => write!(f, "Comment(;{})", s),
            Token::Newline => write!(f, "\\n"),
            Token::Eof => write!(f, "EOF"),
        }
    }
}

pub struct Lexer {
    input: Vec<char>,
    pos: usize,
    line: usize,
}

impl Lexer {
    pub fn new(input: &str) -> Self {
        Lexer {
            input: input.chars().collect(),
            pos: 0,
            line: 1,
        }
    }

    pub fn current_line(&self) -> usize {
        self.line
    }

    fn peek(&self) -> Option<char> {
        if self.pos < self.input.len() {
            Some(self.input[self.pos])
        } else {
            None
        }
    }

    fn peek_ahead(&self, n: usize) -> Option<char> {
        if self.pos + n < self.input.len() {
            Some(self.input[self.pos + n])
        } else {
            None
        }
    }

    fn advance(&mut self) -> Option<char> {
        if self.pos < self.input.len() {
            let ch = self.input[self.pos];
            self.pos += 1;
            if ch == '\n' {
                self.line += 1;
            }
            Some(ch)
        } else {
            None
        }
    }

    fn skip_whitespace(&mut self) {
        while let Some(ch) = self.peek() {
            if ch == ' ' || ch == '\t' || ch == '\r' {
                self.advance();
            } else {
                break;
            }
        }
    }

    fn is_label_start(ch: char) -> bool {
        ch.is_ascii_alphabetic() || ch == '$' || ch == '%' || ch == '_' || ch == '.'
    }

    fn is_label_char(ch: char) -> bool {
        ch.is_ascii_alphanumeric() || ch == '$' || ch == '%' || ch == '_' || ch == '.'
    }

    fn read_identifier(&mut self) -> String {
        let mut result = String::new();

        while let Some(ch) = self.peek() {
            if Self::is_label_char(ch) {
                result.push(ch);
                self.advance();
            } else {
                break;
            }
        }

        result
    }

    fn read_number(&mut self) -> Result<i32, String> {
        let mut result = String::new();
        let mut has_sign = false;

        // Handle optional sign
        if let Some(ch) = self.peek() {
            if ch == '+' || ch == '-' {
                result.push(ch);
                self.advance();
                has_sign = true;
            }
        }

        // Read digits
        let mut has_digits = false;
        while let Some(ch) = self.peek() {
            if ch.is_ascii_digit() {
                result.push(ch);
                self.advance();
                has_digits = true;
            } else {
                break;
            }
        }

        if !has_digits {
            return Err(format!("Expected digit after sign at line {}", self.line));
        }

        result.parse::<i32>()
            .map_err(|e| format!("Failed to parse number '{}': {}", result, e))
    }

    fn read_immediate(&mut self) -> Result<u16, String> {
        // Already consumed '#'
        let mut hex_str = String::new();

        for _ in 0..4 {
            if let Some(ch) = self.peek() {
                if ch.is_ascii_hexdigit() {
                    hex_str.push(ch);
                    self.advance();
                } else {
                    return Err(format!("Expected hex digit in immediate value at line {}", self.line));
                }
            } else {
                return Err(format!("Incomplete immediate value at line {}", self.line));
            }
        }

        u16::from_str_radix(&hex_str, 16)
            .map_err(|e| format!("Failed to parse immediate '{}': {}", hex_str, e))
    }

    fn read_string(&mut self) -> Result<String, String> {
        // Already consumed opening '
        let mut result = String::new();

        loop {
            match self.advance() {
                Some('\'') => {
                    // Check for escaped quote ''
                    if self.peek() == Some('\'') {
                        result.push('\'');
                        self.advance();
                    } else {
                        // End of string
                        return Ok(result);
                    }
                }
                Some(ch) => {
                    result.push(ch);
                }
                None => {
                    return Err(format!("Unterminated string at line {}", self.line));
                }
            }
        }
    }

    fn read_comment(&mut self) -> String {
        let mut result = String::new();

        while let Some(ch) = self.peek() {
            if ch == '\n' {
                break;
            }
            result.push(ch);
            self.advance();
        }

        result
    }

    pub fn next_token(&mut self) -> Result<Token, String> {
        self.skip_whitespace();

        match self.peek() {
            None => Ok(Token::Eof),
            Some('\n') => {
                self.advance();
                Ok(Token::Newline)
            }
            Some(',') => {
                self.advance();
                Ok(Token::Comma)
            }
            Some(':') => {
                self.advance();
                Ok(Token::Colon)
            }
            Some(';') => {
                self.advance();
                let comment = self.read_comment();
                Ok(Token::Comment(comment))
            }
            Some('\'') => {
                self.advance();
                let s = self.read_string()?;
                Ok(Token::String(s))
            }
            Some('#') => {
                self.advance();
                let imm = self.read_immediate()?;
                Ok(Token::Immediate(imm))
            }
            Some('=') => {
                self.advance();
                // Read literal value (can be number, string, or immediate)
                self.skip_whitespace();
                match self.peek() {
                    Some('\'') => {
                        self.advance();
                        let s = self.read_string()?;
                        Ok(Token::Literal(format!("'{}'", s)))
                    }
                    Some('#') => {
                        self.advance();
                        let imm = self.read_immediate()?;
                        Ok(Token::Literal(format!("#{:04X}", imm)))
                    }
                    Some('+') | Some('-') => {
                        let num = self.read_number()?;
                        Ok(Token::Literal(num.to_string()))
                    }
                    Some(ch) if ch.is_ascii_digit() => {
                        let num = self.read_number()?;
                        Ok(Token::Literal(num.to_string()))
                    }
                    _ => Err(format!("Invalid literal at line {}", self.line))
                }
            }
            Some('+') | Some('-') => {
                // Could be a sign for a number
                let num = self.read_number()?;
                Ok(Token::Number(num))
            }
            Some(ch) if ch.is_ascii_digit() => {
                // Check if it's a register (single digit)
                let digit = ch as u8 - b'0';
                if digit <= 7 && self.peek_ahead(1).map_or(true, |c| !c.is_ascii_alphanumeric()) {
                    self.advance();
                    Ok(Token::Register(digit))
                } else {
                    // It's a number
                    let num = self.read_number()?;
                    Ok(Token::Number(num))
                }
            }
            Some(ch) if Self::is_label_start(ch) => {
                let ident = self.read_identifier();

                // Check if it's a register (GRn format)
                if ident.len() == 3 && ident.starts_with("GR") {
                    if let Some(digit_ch) = ident.chars().nth(2) {
                        if let Some(digit) = digit_ch.to_digit(10) {
                            if digit <= 7 {
                                return Ok(Token::Register(digit as u8));
                            }
                        }
                    }
                }

                // Check if it's an instruction (all uppercase)
                if ident.chars().all(|c| c.is_ascii_uppercase()) {
                    // Known instructions
                    match ident.as_str() {
                        "NOP" | "LD" | "ST" | "LAD" | "ADDA" | "SUBA" | "ADDL" | "SUBL" |
                        "MULA" | "DIVA" | "MULL" | "DIVL" | "AND" | "OR" | "XOR" |
                        "CPA" | "CPL" | "SLA" | "SRA" | "SLL" | "SRL" |
                        "JMI" | "JNZ" | "JZE" | "JUMP" | "JPL" | "JOV" |
                        "PUSH" | "POP" | "CALL" | "RET" | "SVC" |
                        "START" | "END" | "DS" | "DC" |
                        "IN" | "OUT" | "RPUSH" | "RPOP" => {
                            return Ok(Token::Instruction(ident));
                        }
                        _ => {}
                    }
                }

                // Otherwise, it's a label
                Ok(Token::Label(ident))
            }
            Some(ch) => {
                Err(format!("Unexpected character '{}' at line {}", ch, self.line))
            }
        }
    }

    pub fn tokenize(&mut self) -> Result<Vec<(Token, usize)>, String> {
        let mut tokens = Vec::new();

        loop {
            let line = self.current_line();
            let token = self.next_token()?;

            if token == Token::Eof {
                tokens.push((token, line));
                break;
            }

            tokens.push((token, line));
        }

        Ok(tokens)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tokenize_simple() {
        let input = "MAIN START\n  LD GR1, DATA\n  RET\nDATA DC 10\n  END";
        let mut lexer = Lexer::new(input);
        let tokens = lexer.tokenize().unwrap();

        assert!(tokens.len() > 0);
    }

    #[test]
    fn test_immediate() {
        let mut lexer = Lexer::new("#1234");
        let token = lexer.next_token().unwrap();
        assert_eq!(token, Token::Immediate(0x1234));
    }

    #[test]
    fn test_string() {
        let mut lexer = Lexer::new("'Hello''World'");
        let token = lexer.next_token().unwrap();
        assert_eq!(token, Token::String("Hello'World".to_string()));
    }

    #[test]
    fn test_register() {
        let mut lexer = Lexer::new("GR3");
        let token = lexer.next_token().unwrap();
        assert_eq!(token, Token::Register(3));
    }
}
