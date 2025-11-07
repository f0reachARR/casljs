// COMET2 Emulator

use std::io::{self, Write};

type Result<T> = std::result::Result<T, String>;

// System call addresses
const SYS_IN: u16 = 0xfff0;
const SYS_OUT: u16 = 0xfff2;
const EXIT_USR: u16 = 0x0000;

// Flag register bits
const FR_SIGN: u16 = 0b001;  // Sign flag (SF)
const FR_ZERO: u16 = 0b010;  // Zero flag (ZF)
const FR_OVERFLOW: u16 = 0b100;  // Overflow flag (OF)

// Stack
const STACK_TOP: u16 = 0xff00;

// Memory size (64K words)
const MEMORY_SIZE: usize = 65536;

pub struct Comet2 {
    // Registers: GR0-GR7, PC, SP, FR
    gr: [u16; 8],
    pc: u16,
    sp: u16,
    fr: u16,

    // Memory (64K words)
    memory: Vec<u16>,

    // Input buffer
    input_lines: Vec<String>,
    current_input: usize,

    // Output buffer (for testing)
    output_buffer: Vec<String>,

    // Options
    quiet: bool,
}

impl Comet2 {
    pub fn new(binary: &[u16], start_address: u16, inputs: &[String], quiet: bool) -> Self {
        let mut memory = vec![0u16; MEMORY_SIZE];

        // Load binary into memory
        for (i, &word) in binary.iter().enumerate() {
            memory[i] = word;
        }

        Self {
            gr: [0; 8],
            pc: start_address,
            sp: STACK_TOP,
            fr: 0,
            memory,
            input_lines: inputs.to_vec(),
            current_input: 0,
            output_buffer: Vec::new(),
            quiet,
        }
    }

    pub fn get_output(&self) -> String {
        self.output_buffer.join("")
    }

    pub fn run(&mut self) -> Result<()> {
        loop {
            // Fetch instruction
            let opcode_word = self.fetch()?;
            let opcode = (opcode_word >> 8) as u8;
            let r = ((opcode_word >> 4) & 0xf) as u8;
            let x = (opcode_word & 0xf) as u8;

            // Execute instruction
            match opcode {
                0x00 => self.nop(),
                0x10 => self.ld_ra(r, x)?,
                0x11 => self.st(r, x)?,
                0x12 => self.lad(r, x)?,
                0x14 => self.ld_rr(r, x),
                0x20 => self.adda_ra(r, x)?,
                0x21 => self.suba_ra(r, x)?,
                0x22 => self.addl_ra(r, x)?,
                0x23 => self.subl_ra(r, x)?,
                0x24 => self.adda_rr(r, x),
                0x25 => self.suba_rr(r, x),
                0x26 => self.addl_rr(r, x),
                0x27 => self.subl_rr(r, x),
                0x28 => self.mula_ra(r, x)?,
                0x29 => self.diva_ra(r, x)?,
                0x2a => self.mull_ra(r, x)?,
                0x2b => self.divl_ra(r, x)?,
                0x2c => self.mula_rr(r, x),
                0x2d => self.diva_rr(r, x)?,
                0x2e => self.mull_rr(r, x),
                0x2f => self.divl_rr(r, x)?,
                0x30 => self.and_ra(r, x)?,
                0x31 => self.or_ra(r, x)?,
                0x32 => self.xor_ra(r, x)?,
                0x34 => self.and_rr(r, x),
                0x35 => self.or_rr(r, x),
                0x36 => self.xor_rr(r, x),
                0x40 => self.cpa_ra(r, x)?,
                0x41 => self.cpl_ra(r, x)?,
                0x44 => self.cpa_rr(r, x),
                0x45 => self.cpl_rr(r, x),
                0x50 => self.sla(r, x)?,
                0x51 => self.sra(r, x)?,
                0x52 => self.sll(r, x)?,
                0x53 => self.srl(r, x)?,
                0x61 => { if self.jmi(x)? { continue; } },
                0x62 => { if self.jnz(x)? { continue; } },
                0x63 => { if self.jze(x)? { continue; } },
                0x64 => { if self.jump(x)? { continue; } },
                0x65 => { if self.jpl(x)? { continue; } },
                0x66 => { if self.jov(x)? { continue; } },
                0x70 => self.push(x)?,
                0x71 => self.pop(r),
                0x80 => self.call(x)?,
                0x81 => {
                    if !self.ret() {
                        // Program terminated
                        return Ok(());
                    }
                },
                0xf0 => {
                    // SVC
                    let svc_code = self.fetch()?;
                    self.svc(svc_code)?;
                    if svc_code == 0 || svc_code == 1 || svc_code == 2 || svc_code == 3 {
                        // Program terminated
                        return Ok(());
                    }
                },
                _ => return Err(format!("Unknown opcode: 0x{:02X} at PC=0x{:04X}", opcode, self.pc - 1)),
            }
        }
    }

    fn fetch(&mut self) -> Result<u16> {
        if self.pc as usize >= MEMORY_SIZE {
            return Err(format!("PC out of bounds: 0x{:04X}", self.pc));
        }
        let word = self.memory[self.pc as usize];
        self.pc += 1;
        Ok(word)
    }

    fn effective_address(&mut self, x: u8) -> Result<u16> {
        let adr = self.fetch()?;
        if x == 0 {
            Ok(adr)
        } else {
            Ok(adr.wrapping_add(self.gr[x as usize]))
        }
    }

    fn set_flags_arithmetic(&mut self, result: i16) {
        self.fr = 0;
        if result < 0 {
            self.fr |= FR_SIGN;
        } else if result == 0 {
            self.fr |= FR_ZERO;
        }
    }

    fn set_flags_logical(&mut self, result: u16) {
        self.fr = 0;
        if result == 0 {
            self.fr |= FR_ZERO;
        } else if (result & 0x8000) != 0 {
            self.fr |= FR_SIGN;
        }
    }

    fn set_overflow(&mut self) {
        self.fr |= FR_OVERFLOW;
    }

    // Instructions

    fn nop(&mut self) {
        // Do nothing
    }

    fn ld_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        self.gr[r as usize] = self.memory[ea as usize];
        Ok(())
    }

    fn ld_rr(&mut self, r1: u8, r2: u8) {
        self.gr[r1 as usize] = self.gr[r2 as usize];
    }

    fn st(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        self.memory[ea as usize] = self.gr[r as usize];
        Ok(())
    }

    fn lad(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        self.gr[r as usize] = ea;
        Ok(())
    }

    fn adda_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize] as i16;
        let gr_val = self.gr[r as usize] as i16;

        let (result, overflow) = gr_val.overflowing_add(val);
        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if overflow {
            self.set_overflow();
        }
        Ok(())
    }

    fn adda_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize] as i16;
        let val2 = self.gr[r2 as usize] as i16;

        let (result, overflow) = val1.overflowing_add(val2);
        self.gr[r1 as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if overflow {
            self.set_overflow();
        }
    }

    fn suba_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize] as i16;
        let gr_val = self.gr[r as usize] as i16;

        let (result, overflow) = gr_val.overflowing_sub(val);
        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if overflow {
            self.set_overflow();
        }
        Ok(())
    }

    fn suba_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize] as i16;
        let val2 = self.gr[r2 as usize] as i16;

        let (result, overflow) = val1.overflowing_sub(val2);
        self.gr[r1 as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if overflow {
            self.set_overflow();
        }
    }

    fn addl_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let gr_val = self.gr[r as usize];

        let result = gr_val.wrapping_add(val);
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        if (gr_val as u32 + val as u32) > 0xFFFF {
            self.set_overflow();
        }
        Ok(())
    }

    fn addl_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize];
        let val2 = self.gr[r2 as usize];

        let result = val1.wrapping_add(val2);
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
        if (val1 as u32 + val2 as u32) > 0xFFFF {
            self.set_overflow();
        }
    }

    fn subl_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let gr_val = self.gr[r as usize];

        let result = gr_val.wrapping_sub(val);
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        if gr_val < val {
            self.set_overflow();
        }
        Ok(())
    }

    fn subl_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize];
        let val2 = self.gr[r2 as usize];

        let result = val1.wrapping_sub(val2);
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
        if val1 < val2 {
            self.set_overflow();
        }
    }

    fn mula_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize] as i16;
        let gr_val = self.gr[r as usize] as i16;

        let result32 = (gr_val as i32) * (val as i32);
        let result = result32 as i16;
        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if result32 < -32768 || result32 > 32767 {
            self.set_overflow();
        }
        Ok(())
    }

    fn mula_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize] as i16;
        let val2 = self.gr[r2 as usize] as i16;

        let result32 = (val1 as i32) * (val2 as i32);
        let result = result32 as i16;
        self.gr[r1 as usize] = result as u16;
        self.set_flags_arithmetic(result);
        if result32 < -32768 || result32 > 32767 {
            self.set_overflow();
        }
    }

    fn diva_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let divisor = self.memory[ea as usize] as i16;
        let dividend = self.gr[r as usize] as i16;

        if divisor == 0 {
            // Division by zero - behavior matches original (sets result to dividend)
            self.set_flags_arithmetic(dividend);
            return Ok(());
        }

        let result = dividend / divisor;
        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        Ok(())
    }

    fn diva_rr(&mut self, r1: u8, r2: u8) -> Result<()> {
        let dividend = self.gr[r1 as usize] as i16;
        let divisor = self.gr[r2 as usize] as i16;

        if divisor == 0 {
            self.set_flags_arithmetic(dividend);
            return Ok(());
        }

        let result = dividend / divisor;
        self.gr[r1 as usize] = result as u16;
        self.set_flags_arithmetic(result);
        Ok(())
    }

    fn mull_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let gr_val = self.gr[r as usize];

        let result32 = (gr_val as u32) * (val as u32);
        let result = result32 as u16;
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        if result32 > 0xFFFF {
            self.set_overflow();
        }
        Ok(())
    }

    fn mull_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize];
        let val2 = self.gr[r2 as usize];

        let result32 = (val1 as u32) * (val2 as u32);
        let result = result32 as u16;
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
        if result32 > 0xFFFF {
            self.set_overflow();
        }
    }

    fn divl_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let divisor = self.memory[ea as usize];
        let dividend = self.gr[r as usize];

        if divisor == 0 {
            self.set_flags_logical(dividend);
            return Ok(());
        }

        let result = dividend / divisor;
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn divl_rr(&mut self, r1: u8, r2: u8) -> Result<()> {
        let dividend = self.gr[r1 as usize];
        let divisor = self.gr[r2 as usize];

        if divisor == 0 {
            self.set_flags_logical(dividend);
            return Ok(());
        }

        let result = dividend / divisor;
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn and_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let result = self.gr[r as usize] & val;
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn and_rr(&mut self, r1: u8, r2: u8) {
        let result = self.gr[r1 as usize] & self.gr[r2 as usize];
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
    }

    fn or_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let result = self.gr[r as usize] | val;
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn or_rr(&mut self, r1: u8, r2: u8) {
        let result = self.gr[r1 as usize] | self.gr[r2 as usize];
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
    }

    fn xor_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let result = self.gr[r as usize] ^ val;
        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn xor_rr(&mut self, r1: u8, r2: u8) {
        let result = self.gr[r1 as usize] ^ self.gr[r2 as usize];
        self.gr[r1 as usize] = result;
        self.set_flags_logical(result);
    }

    fn cpa_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize] as i16;
        let gr_val = self.gr[r as usize] as i16;

        let result = gr_val.wrapping_sub(val);
        self.set_flags_arithmetic(result);
        Ok(())
    }

    fn cpa_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize] as i16;
        let val2 = self.gr[r2 as usize] as i16;

        let result = val1.wrapping_sub(val2);
        self.set_flags_arithmetic(result);
    }

    fn cpl_ra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let val = self.memory[ea as usize];
        let gr_val = self.gr[r as usize];

        let result = gr_val.wrapping_sub(val);
        self.set_flags_logical(result);
        Ok(())
    }

    fn cpl_rr(&mut self, r1: u8, r2: u8) {
        let val1 = self.gr[r1 as usize];
        let val2 = self.gr[r2 as usize];

        let result = val1.wrapping_sub(val2);
        self.set_flags_logical(result);
    }

    fn sla(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let shift_bits = self.memory[ea as usize] as i16;
        let val = self.gr[r as usize] as i16;

        let result = if shift_bits >= 0 {
            val << shift_bits
        } else {
            val >> (-shift_bits)
        };

        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        Ok(())
    }

    fn sra(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let shift_bits = self.memory[ea as usize] as i16;
        let val = self.gr[r as usize] as i16;

        let result = if shift_bits >= 0 {
            val >> shift_bits
        } else {
            val << (-shift_bits)
        };

        self.gr[r as usize] = result as u16;
        self.set_flags_arithmetic(result);
        Ok(())
    }

    fn sll(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let shift_bits = self.memory[ea as usize] as i16;
        let val = self.gr[r as usize];

        let result = if shift_bits >= 0 {
            val << shift_bits
        } else {
            val >> (-shift_bits)
        };

        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn srl(&mut self, r: u8, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        let shift_bits = self.memory[ea as usize] as i16;
        let val = self.gr[r as usize];

        let result = if shift_bits >= 0 {
            val >> shift_bits
        } else {
            val << (-shift_bits)
        };

        self.gr[r as usize] = result;
        self.set_flags_logical(result);
        Ok(())
    }

    fn jmi(&mut self, x: u8) -> Result<bool> {
        if (self.fr & FR_SIGN) != 0 {
            let ea = self.effective_address(x)?;
            self.pc = ea;
            Ok(true)
        } else {
            self.pc += 1; // Skip address word
            Ok(false)
        }
    }

    fn jnz(&mut self, x: u8) -> Result<bool> {
        if (self.fr & FR_ZERO) == 0 {
            let ea = self.effective_address(x)?;
            self.pc = ea;
            Ok(true)
        } else {
            self.pc += 1;
            Ok(false)
        }
    }

    fn jze(&mut self, x: u8) -> Result<bool> {
        if (self.fr & FR_ZERO) != 0 {
            let ea = self.effective_address(x)?;
            self.pc = ea;
            Ok(true)
        } else {
            self.pc += 1;
            Ok(false)
        }
    }

    fn jump(&mut self, x: u8) -> Result<bool> {
        let ea = self.effective_address(x)?;
        self.pc = ea;
        Ok(true)
    }

    fn jpl(&mut self, x: u8) -> Result<bool> {
        if (self.fr & (FR_SIGN | FR_ZERO)) == 0 {
            let ea = self.effective_address(x)?;
            self.pc = ea;
            Ok(true)
        } else {
            self.pc += 1;
            Ok(false)
        }
    }

    fn jov(&mut self, x: u8) -> Result<bool> {
        if (self.fr & FR_OVERFLOW) != 0 {
            let ea = self.effective_address(x)?;
            self.pc = ea;
            Ok(true)
        } else {
            self.pc += 1;
            Ok(false)
        }
    }

    fn push(&mut self, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        self.sp = self.sp.wrapping_sub(1);
        self.memory[self.sp as usize] = ea;  // Push the effective address itself, not memory[ea]
        Ok(())
    }

    fn pop(&mut self, r: u8) {
        self.gr[r as usize] = self.memory[self.sp as usize];
        self.sp = self.sp.wrapping_add(1);
    }

    fn call(&mut self, x: u8) -> Result<()> {
        let ea = self.effective_address(x)?;
        self.sp = self.sp.wrapping_sub(1);
        self.memory[self.sp as usize] = self.pc;
        self.pc = ea;
        Ok(())
    }

    fn ret(&mut self) -> bool {
        if self.sp >= STACK_TOP {
            // Stack underflow - program terminated
            return false;
        }
        self.pc = self.memory[self.sp as usize];
        self.sp = self.sp.wrapping_add(1);
        true
    }

    fn svc(&mut self, code: u16) -> Result<()> {
        match code {
            0 => {
                // Normal termination
                if !self.quiet {
                    println!("Program terminated normally.");
                }
            }
            1 => {
                // Overflow error
                if !self.quiet {
                    println!("***** Run-Time Error : Overflow *****");
                }
            }
            2 => {
                // Zero divide error
                if !self.quiet {
                    println!("***** Run-Time Error : Zero-Divide *****");
                }
            }
            3 => {
                // Range over error
                if !self.quiet {
                    println!("***** Run-Time Error : Range-Over in Array Index *****");
                }
            }
            SYS_IN => {
                // IN: Get buf_addr from GR1, len_addr from GR2
                let buf_addr = self.gr[1];
                let len_addr = self.gr[2];

                // Read input line
                if self.current_input < self.input_lines.len() {
                    let line = &self.input_lines[self.current_input];
                    self.current_input += 1;

                    if !self.quiet {
                        let output_line = format!("IN> {}\n", line);
                        print!("{}", output_line);
                        io::stdout().flush().ok();
                        self.output_buffer.push(output_line);
                    }

                    // Store characters in buffer
                    let max_len = self.memory[len_addr as usize] as usize;
                    let chars: Vec<char> = line.chars().collect();
                    let copy_len = chars.len().min(max_len);

                    for (i, &ch) in chars.iter().take(copy_len).enumerate() {
                        self.memory[(buf_addr as usize) + i] = ch as u16;
                    }

                    // Update length
                    self.memory[len_addr as usize] = copy_len as u16;
                } else {
                    // No more input, set length to 0
                    self.memory[len_addr as usize] = 0;
                }
            }
            SYS_OUT => {
                // OUT: Get buf_addr from GR1, len_addr from GR2
                let buf_addr = self.gr[1];
                let len_addr = self.gr[2];

                let len = self.memory[len_addr as usize] as usize;
                let mut output_line = String::new();
                if !self.quiet {
                    print!("OUT> ");
                    output_line.push_str("OUT> ");
                }

                for i in 0..len {
                    let ch = self.memory[(buf_addr as usize) + i];
                    if ch == 0 {
                        break;
                    }
                    let c = (ch as u8) as char;
                    print!("{}", c);
                    output_line.push(c);
                }
                println!();
                output_line.push('\n');
                if !self.quiet {
                    self.output_buffer.push(output_line);
                }
                io::stdout().flush().ok();
            }
            _ => {
                return Err(format!("Unknown SVC code: {}", code));
            }
        }
        Ok(())
    }
}
