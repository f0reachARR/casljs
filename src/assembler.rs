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
                        address += values.len() as u16;
                    }
                    Instruction::In(_, _) => {
                        // IN macro expands to multiple instructions
                        // For now, estimate size (actual expansion in pass2)
                        address += 8; // Approximate size
                    }
                    Instruction::Out(_, _) => {
                        // OUT macro expands to multiple instructions
                        address += 8; // Approximate size
                    }
                    Instruction::Rpush => {
                        // RPUSH macro: PUSH all registers
                        address += 7; // PUSH for GR1-GR7
                    }
                    Instruction::Rpop => {
                        // RPOP macro: POP all registers in reverse
                        address += 7; // POP for GR7-GR1
                    }
                    Instruction::NoOperand(_) => {
                        address += 1; // 1 word
                    }
                    Instruction::OneReg(_, _) => {
                        address += 1; // 1 word (POP)
                    }
                    Instruction::Addr(_, _, _) => {
                        address += 2; // 2 words
                    }
                    Instruction::RegAddr(inst, _, _, _) => {
                        address += 2; // 2 words
                    }
                    Instruction::TwoReg(_, _, _) => {
                        address += 1; // 1 word
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
                        in_program = false;
                        self.current_scope.clear();
                    }
                    Instruction::Ds(count) => {
                        // Allocate uninitialized memory
                        for _ in 0..*count {
                            binary.push(0);
                        }
                    }
                    Instruction::Dc(values) => {
                        for value in values {
                            let word = self.resolve_dc_value(value, line.line_number)?;
                            binary.push(word);
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
                        let code = self.opcode_addr(op);
                        let xr = index.as_ref().map_or(0, |r| r.to_u8());
                        binary.push((code << 8) | (xr as u16));
                        let addr_val = self.resolve_address(addr, line.line_number)?;
                        binary.push(addr_val);
                    }
                    Instruction::RegAddr(op, reg, addr, index) => {
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
                    Instruction::In(_, _) => {
                        return Err(format!(
                            "Line {}: IN macro not yet implemented",
                            line.line_number
                        ));
                    }
                    Instruction::Out(_, _) => {
                        return Err(format!(
                            "Line {}: OUT macro not yet implemented",
                            line.line_number
                        ));
                    }
                    Instruction::Rpush => {
                        return Err(format!(
                            "Line {}: RPUSH macro not yet implemented",
                            line.line_number
                        ));
                    }
                    Instruction::Rpop => {
                        return Err(format!(
                            "Line {}: RPOP macro not yet implemented",
                            line.line_number
                        ));
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
            DcValue::String(s) => {
                if s.len() != 1 {
                    return Err(format!(
                        "Line {}: DC string must be single character or use multiple DC values",
                        line_number
                    ));
                }
                Ok(s.chars().next().unwrap() as u16)
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
            Address::Literal(_) => {
                // TODO: Implement literal expansion
                Err(format!(
                    "Line {}: Literal addressing not yet implemented",
                    line_number
                ))
            }
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
