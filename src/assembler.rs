// Code generator for CASL2
// Two-pass assembler

use crate::ast::*;
use std::collections::HashMap;

type Result<T> = std::result::Result<T, String>;

// Symbol table entry
#[derive(Debug, Clone)]
struct Symbol {
    address: u16,
    line_number: usize,
}

// Memory entry for pass 1
#[derive(Debug, Clone)]
enum MemoryEntry {
    Word(u16),
    LabelRef(String),  // Reference to a label to be resolved in pass 2
}

pub struct Assembler {
    program: Program,
    symbols: HashMap<String, Symbol>,
    memory: Vec<MemoryEntry>,
    current_scope: String,
    literal_counter: usize,
    start_address: u16,
    literals: HashMap<String, String>,  // literal_value -> label
}

impl Assembler {
    pub fn new(program: Program) -> Self {
        Self {
            program,
            symbols: HashMap::new(),
            memory: Vec::new(),
            current_scope: String::new(),
            literal_counter: 0,
            start_address: 0,
            literals: HashMap::new(),
        }
    }

    pub fn assemble(mut self) -> Result<(Vec<u16>, u16, u16)> {
        // Pass 1: Build symbol table
        self.pass1()?;

        // Pass 2: Generate machine code
        let binary = self.pass2()?;

        let address_max = self.memory.len() as u16;

        Ok((binary, self.start_address, address_max))
    }

    fn pass1(&mut self) -> Result<()> {
        let mut address: u16 = 0;
        let mut in_program = false;
        let mut start_label: Option<String> = None;
        let mut scope_literals: Vec<String> = Vec::new();

        for line in &self.program.lines {
            // Handle START specially - set scope before adding label
            if let Some(Instruction::Start(entry)) = &line.instruction {
                if in_program {
                    return Err(format!(
                        "Line {}: Nested START not allowed",
                        line.line_number
                    ));
                }
                in_program = true;

                // Set scope to the label of this START
                if let Some(ref label) = line.label {
                    self.current_scope = label.clone();
                } else {
                    return Err(format!(
                        "Line {}: START must have a label",
                        line.line_number
                    ));
                }

                // Set start address if entry point is specified
                if entry.is_some() || start_label.is_none() {
                    start_label = Some(self.current_scope.clone());
                }
            }

            // Process label
            if let Some(ref label) = line.label {
                let scoped_label = format!("{}:{}", self.current_scope, label);
                self.symbols.insert(
                    scoped_label,
                    Symbol {
                        address,
                        line_number: line.line_number,
                    },
                );
            }

            // Process instruction
            if let Some(ref inst) = line.instruction {
                match inst {
                    Instruction::Start(_) => {
                        // Already handled above
                    }
                    Instruction::End => {
                        if !in_program {
                            return Err(format!(
                                "Line {}: END without START",
                                line.line_number
                            ));
                        }

                        // Place literals before END
                        for lit_val in &scope_literals {
                            let lit_label = format!("{}:_LIT{}", self.current_scope, self.literal_counter);
                            self.literal_counter += 1;
                            self.literals.insert(lit_val.clone(), lit_label.clone());

                            self.symbols.insert(
                                lit_label,
                                Symbol {
                                    address,
                                    line_number: line.line_number,
                                },
                            );
                            address += 1; // Each literal takes 1 word
                        }
                        scope_literals.clear();

                        in_program = false;
                        self.current_scope.clear();
                    }
                    Instruction::Ds(count) => {
                        if *count < 0 {
                            return Err(format!(
                                "Line {}: DS count must be non-negative",
                                line.line_number
                            ));
                        }
                        address += *count as u16;
                    }
                    Instruction::Dc(values) => {
                        // Calculate actual size: strings expand to multiple words
                        for value in values {
                            match value {
                                DcValue::String(s) => {
                                    address += s.len() as u16;
                                }
                                _ => {
                                    address += 1;
                                }
                            }
                        }
                    }
                    Instruction::In(_, _) => {
                        // IN macro: PUSH GR1, PUSH GR2, LAD GR1, LAD GR2, SVC, POP GR2, POP GR1 = 12 words
                        address += 12;
                    }
                    Instruction::Out(_, _) => {
                        // OUT macro: PUSH GR1, PUSH GR2, LAD GR1, LAD GR2, SVC, POP GR2, POP GR1 = 12 words
                        address += 12;
                    }
                    Instruction::Rpush => {
                        // RPUSH macro: PUSH for GR1-GR7
                        address += 14; // 7 PUSH instructions, each 2 words
                    }
                    Instruction::Rpop => {
                        // RPOP macro: POP for GR7-GR1
                        address += 7; // 7 POP instructions, each 1 word
                    }
                    Instruction::NoOperand(_) => {
                        address += 1;
                    }
                    Instruction::OneReg(_, _) => {
                        address += 1;
                    }
                    Instruction::Addr(_, addr, _) => {
                        if let Address::Literal(lit) = addr {
                            if !scope_literals.contains(lit) {
                                scope_literals.push(lit.clone());
                            }
                        }
                        address += 2;
                    }
                    Instruction::RegAddr(_, _, addr, _) => {
                        if let Address::Literal(lit) = addr {
                            if !scope_literals.contains(lit) {
                                scope_literals.push(lit.clone());
                            }
                        }
                        address += 2;
                    }
                    Instruction::TwoReg(_, _, _) => {
                        address += 1;
                    }
                }
            }
        }

        if in_program {
            return Err("Missing END instruction".to_string());
        }

        // Resolve start address
        if let Some(label) = start_label {
            let scoped = format!("{}:{}", label, label);
            if let Some(sym) = self.symbols.get(&scoped) {
                self.start_address = sym.address;
            } else {
                return Err(format!("Start label '{}' not found", label));
            }
        }

        Ok(())
    }

    fn pass2(&mut self) -> Result<Vec<u16>> {
        let mut binary: Vec<u16> = Vec::new();
        let mut in_program = false;
        let mut scope_literals: Vec<String> = Vec::new();

        for line in &self.program.lines {
            if let Some(ref inst) = line.instruction {
                match inst {
                    Instruction::Start(_) => {
                        in_program = true;
                        if let Some(ref label) = line.label {
                            self.current_scope = label.clone();
                        }
                    }
                    Instruction::End => {
                        // Generate literals before END
                        for lit_val in &scope_literals {
                            let word = self.parse_literal_value(lit_val)?;
                            binary.push(word);
                        }
                        scope_literals.clear();

                        in_program = false;
                        self.current_scope.clear();
                    }
                    Instruction::Ds(count) => {
                        for _ in 0..*count {
                            binary.push(0);
                        }
                    }
                    Instruction::Dc(values) => {
                        for value in values {
                            match value {
                                DcValue::String(s) => {
                                    // Each character becomes a separate word
                                    for ch in s.chars() {
                                        binary.push(ch as u16);
                                    }
                                }
                                _ => {
                                    let word = self.resolve_dc_value(value, line.line_number)?;
                                    binary.push(word);
                                }
                            }
                        }
                    }
                    Instruction::NoOperand(op) => {
                        let code = self.opcode_no_operand(op);
                        binary.push(code << 8);
                    }
                    Instruction::OneReg(op, reg) => {
                        let code = self.opcode_one_reg(op);
                        let r = reg.to_u8();
                        binary.push((code << 8) | (r << 4) as u16);
                    }
                    Instruction::Addr(op, addr, index) => {
                        if let Address::Literal(lit) = addr {
                            if !scope_literals.contains(lit) {
                                scope_literals.push(lit.clone());
                            }
                        }
                        let code = self.opcode_addr(op);
                        let xr = index.as_ref().map_or(0, |r| r.to_u8());
                        binary.push((code << 8) | (xr as u16));
                        let addr_val = self.resolve_address(addr, line.line_number)?;
                        binary.push(addr_val);
                    }
                    Instruction::RegAddr(op, reg, addr, index) => {
                        if let Address::Literal(lit) = addr {
                            if !scope_literals.contains(lit) {
                                scope_literals.push(lit.clone());
                            }
                        }
                        let code = self.opcode_reg_addr(op);
                        let r = reg.to_u8();
                        let xr = index.as_ref().map_or(0, |r| r.to_u8());
                        binary.push((code << 8) | ((r << 4) as u16) | (xr as u16));
                        let addr_val = self.resolve_address(addr, line.line_number)?;
                        binary.push(addr_val);
                    }
                    Instruction::TwoReg(op, r1, r2) => {
                        let code = self.opcode_two_reg(op);
                        let reg1 = r1.to_u8();
                        let reg2 = r2.to_u8();
                        binary.push((code << 8) | ((reg1 << 4) as u16) | (reg2 as u16));
                    }
                    Instruction::In(buf_label, len_label) => {
                        // IN macro: PUSH GR1, PUSH GR2, LAD GR1=buf, LAD GR2=len, SVC
                        let buf_addr = self.resolve_label(buf_label)?;
                        let len_addr = self.resolve_label(len_label)?;

                        // PUSH 0,GR1 - save GR1
                        binary.push(0x7010);
                        binary.push(0);

                        // PUSH 0,GR2 - save GR2
                        binary.push(0x7020);
                        binary.push(0);

                        // LAD GR1,buf_addr
                        binary.push(0x1210);
                        binary.push(buf_addr);

                        // LAD GR2,len_addr
                        binary.push(0x1220);
                        binary.push(len_addr);

                        // SVC SYS_IN (0xFFF0)
                        binary.push(0xF000);
                        binary.push(0xFFF0);

                        // POP GR2 - restore GR2
                        binary.push(0x7120);

                        // POP GR1 - restore GR1
                        binary.push(0x7110);
                    }
                    Instruction::Out(buf_label, len_label) => {
                        // OUT macro: PUSH GR1, PUSH GR2, LAD GR1=buf, LAD GR2=len, SVC
                        let buf_addr = self.resolve_label(buf_label)?;
                        let len_addr = self.resolve_label(len_label)?;

                        // PUSH 0,GR1 - save GR1
                        binary.push(0x7010);
                        binary.push(0);

                        // PUSH 0,GR2 - save GR2
                        binary.push(0x7020);
                        binary.push(0);

                        // LAD GR1,buf_addr
                        binary.push(0x1210);
                        binary.push(buf_addr);

                        // LAD GR2,len_addr
                        binary.push(0x1220);
                        binary.push(len_addr);

                        // SVC SYS_OUT (0xFFF2)
                        binary.push(0xF000);
                        binary.push(0xFFF2);

                        // POP GR2 - restore GR2
                        binary.push(0x7120);

                        // POP GR1 - restore GR1
                        binary.push(0x7110);
                    }
                    Instruction::Rpush => {
                        // RPUSH: PUSH 0,GR1; PUSH 0,GR2; ... PUSH 0,GR7
                        for reg in 1..=7 {
                            binary.push(0x7000 | (reg << 4)); // PUSH with register
                            binary.push(0);
                        }
                    }
                    Instruction::Rpop => {
                        // RPOP: POP GR7; POP GR6; ... POP GR1
                        for reg in (1..=7).rev() {
                            binary.push(0x7100 | (reg << 4)); // POP with register
                        }
                    }
                }
            }
        }

        Ok(binary)
    }

    fn resolve_dc_value(&self, value: &DcValue, line_number: usize) -> Result<u16> {
        match value {
            DcValue::Number(n) => Ok(*n as u16),
            DcValue::Immediate(imm) => Ok(*imm),
            DcValue::String(_) => {
                // String values should be handled separately in pass2
                Err(format!(
                    "Line {}: Internal error: String should not reach resolve_dc_value",
                    line_number
                ))
            }
            DcValue::Label(label) => {
                let scoped = format!("{}:{}", self.current_scope, label);
                if let Some(sym) = self.symbols.get(&scoped) {
                    Ok(sym.address)
                } else {
                    Err(format!(
                        "Line {}: Undefined label '{}'",
                        line_number, label
                    ))
                }
            }
        }
    }

    fn resolve_address(&self, addr: &Address, line_number: usize) -> Result<u16> {
        match addr {
            Address::Number(n) => Ok(*n as u16),
            Address::Immediate(imm) => Ok(*imm),
            Address::Label(label) => {
                let scoped = format!("{}:{}", self.current_scope, label);
                if let Some(sym) = self.symbols.get(&scoped) {
                    Ok(sym.address)
                } else {
                    Err(format!(
                        "Line {}: Undefined label '{}'",
                        line_number, label
                    ))
                }
            }
            Address::Literal(lit) => {
                // Resolve literal to its label
                if let Some(lit_label) = self.literals.get(lit) {
                    if let Some(sym) = self.symbols.get(lit_label) {
                        Ok(sym.address)
                    } else {
                        Err(format!(
                            "Line {}: Internal error: literal label not found",
                            line_number
                        ))
                    }
                } else {
                    Err(format!(
                        "Line {}: Internal error: literal not registered",
                        line_number
                    ))
                }
            }
        }
    }

    fn resolve_label(&self, label: &str) -> Result<u16> {
        let scoped = format!("{}:{}", self.current_scope, label);
        if let Some(sym) = self.symbols.get(&scoped) {
            Ok(sym.address)
        } else {
            Err(format!("Undefined label '{}'", label))
        }
    }

    fn parse_literal_value(&self, lit: &str) -> Result<u16> {
        // Parse literal value (e.g., "10", "'A'", "#FFFF")
        if lit.starts_with('\'') && lit.ends_with('\'') {
            // String literal
            let s = &lit[1..lit.len()-1];
            if s.len() != 1 {
                return Err(format!("Literal string must be single character: {}", lit));
            }
            Ok(s.chars().next().unwrap() as u16)
        } else if lit.starts_with('#') {
            // Hex immediate
            u16::from_str_radix(&lit[1..], 16)
                .map_err(|_| format!("Invalid hex literal: {}", lit))
        } else {
            // Decimal number
            lit.parse::<i32>()
                .map(|n| n as u16)
                .map_err(|_| format!("Invalid numeric literal: {}", lit))
        }
    }

    // Opcode mappings
    fn opcode_no_operand(&self, op: &NoOperandInst) -> u16 {
        match op {
            NoOperandInst::Nop => 0x00,
            NoOperandInst::Ret => 0x81,
        }
    }

    fn opcode_one_reg(&self, op: &OneRegInst) -> u16 {
        match op {
            OneRegInst::Pop => 0x71,
        }
    }

    fn opcode_addr(&self, op: &AddrInst) -> u16 {
        match op {
            AddrInst::Jmi => 0x61,
            AddrInst::Jnz => 0x62,
            AddrInst::Jze => 0x63,
            AddrInst::Jump => 0x64,
            AddrInst::Jpl => 0x65,
            AddrInst::Jov => 0x66,
            AddrInst::Push => 0x70,
            AddrInst::Call => 0x80,
            AddrInst::Svc => 0xf0,
        }
    }

    fn opcode_reg_addr(&self, op: &RegAddrInst) -> u16 {
        match op {
            RegAddrInst::Ld => 0x10,
            RegAddrInst::St => 0x11,
            RegAddrInst::Lad => 0x12,
            RegAddrInst::Adda => 0x20,
            RegAddrInst::Suba => 0x21,
            RegAddrInst::Addl => 0x22,
            RegAddrInst::Subl => 0x23,
            RegAddrInst::Mula => 0x28,
            RegAddrInst::Diva => 0x29,
            RegAddrInst::Mull => 0x2a,
            RegAddrInst::Divl => 0x2b,
            RegAddrInst::And => 0x30,
            RegAddrInst::Or => 0x31,
            RegAddrInst::Xor => 0x32,
            RegAddrInst::Cpa => 0x40,
            RegAddrInst::Cpl => 0x41,
            RegAddrInst::Sla => 0x50,
            RegAddrInst::Sra => 0x51,
            RegAddrInst::Sll => 0x52,
            RegAddrInst::Srl => 0x53,
        }
    }

    fn opcode_two_reg(&self, op: &TwoRegInst) -> u16 {
        match op {
            TwoRegInst::Ld => 0x14,
            TwoRegInst::Adda => 0x24,
            TwoRegInst::Suba => 0x25,
            TwoRegInst::Addl => 0x26,
            TwoRegInst::Subl => 0x27,
            TwoRegInst::Mula => 0x2c,
            TwoRegInst::Diva => 0x2d,
            TwoRegInst::Mull => 0x2e,
            TwoRegInst::Divl => 0x2f,
            TwoRegInst::And => 0x34,
            TwoRegInst::Or => 0x35,
            TwoRegInst::Xor => 0x36,
            TwoRegInst::Cpa => 0x44,
            TwoRegInst::Cpl => 0x45,
        }
    }
}
