// Abstract Syntax Tree for CASL2

#[derive(Debug, Clone, PartialEq)]
pub struct Program {
    pub lines: Vec<Line>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Line {
    pub label: Option<String>,
    pub instruction: Option<Instruction>,
    pub line_number: usize,
}

#[derive(Debug, Clone, PartialEq)]
pub enum Instruction {
    // Machine instructions
    NoOperand(NoOperandInst),
    OneReg(OneRegInst, Register),
    RegAddr(RegAddrInst, Register, Address, Option<Register>),
    Addr(AddrInst, Address, Option<Register>),
    TwoReg(TwoRegInst, Register, Register),

    // Pseudo instructions
    Start(Option<String>),  // Optional entry label
    End,
    Ds(i32),                // Count
    Dc(Vec<DcValue>),       // Values

    // Macros
    In(String, String),     // buffer_label, length_label
    Out(String, String),
    Rpush,
    Rpop,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum NoOperandInst {
    Nop,
    Ret,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum OneRegInst {
    Pop,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum RegAddrInst {
    Ld,
    St,
    Lad,
    Adda,
    Suba,
    Addl,
    Subl,
    Mula,
    Diva,
    Mull,
    Divl,
    And,
    Or,
    Xor,
    Cpa,
    Cpl,
    Sla,
    Sra,
    Sll,
    Srl,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum AddrInst {
    Jmi,
    Jnz,
    Jze,
    Jump,
    Jpl,
    Jov,
    Push,
    Call,
    Svc,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TwoRegInst {
    Ld,
    Adda,
    Suba,
    Addl,
    Subl,
    Mula,
    Diva,
    Mull,
    Divl,
    And,
    Or,
    Xor,
    Cpa,
    Cpl,
}

#[derive(Debug, Clone, PartialEq)]
pub enum Register {
    Gr0,
    Gr1,
    Gr2,
    Gr3,
    Gr4,
    Gr5,
    Gr6,
    Gr7,
}

impl Register {
    pub fn from_u8(n: u8) -> Option<Self> {
        match n {
            0 => Some(Register::Gr0),
            1 => Some(Register::Gr1),
            2 => Some(Register::Gr2),
            3 => Some(Register::Gr3),
            4 => Some(Register::Gr4),
            5 => Some(Register::Gr5),
            6 => Some(Register::Gr6),
            7 => Some(Register::Gr7),
            _ => None,
        }
    }

    pub fn to_u8(&self) -> u8 {
        match self {
            Register::Gr0 => 0,
            Register::Gr1 => 1,
            Register::Gr2 => 2,
            Register::Gr3 => 3,
            Register::Gr4 => 4,
            Register::Gr5 => 5,
            Register::Gr6 => 6,
            Register::Gr7 => 7,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum Address {
    Label(String),
    Number(i32),
    Immediate(u16),
    Literal(String),
}

#[derive(Debug, Clone, PartialEq)]
pub enum DcValue {
    Number(i32),
    String(String),
    Label(String),
    Immediate(u16),
}

impl RegAddrInst {
    pub fn from_str(s: &str) -> Option<Self> {
        match s {
            "LD" => Some(RegAddrInst::Ld),
            "ST" => Some(RegAddrInst::St),
            "LAD" => Some(RegAddrInst::Lad),
            "ADDA" => Some(RegAddrInst::Adda),
            "SUBA" => Some(RegAddrInst::Suba),
            "ADDL" => Some(RegAddrInst::Addl),
            "SUBL" => Some(RegAddrInst::Subl),
            "MULA" => Some(RegAddrInst::Mula),
            "DIVA" => Some(RegAddrInst::Diva),
            "MULL" => Some(RegAddrInst::Mull),
            "DIVL" => Some(RegAddrInst::Divl),
            "AND" => Some(RegAddrInst::And),
            "OR" => Some(RegAddrInst::Or),
            "XOR" => Some(RegAddrInst::Xor),
            "CPA" => Some(RegAddrInst::Cpa),
            "CPL" => Some(RegAddrInst::Cpl),
            "SLA" => Some(RegAddrInst::Sla),
            "SRA" => Some(RegAddrInst::Sra),
            "SLL" => Some(RegAddrInst::Sll),
            "SRL" => Some(RegAddrInst::Srl),
            _ => None,
        }
    }

    pub fn can_be_two_reg(&self) -> bool {
        matches!(self,
            RegAddrInst::Ld | RegAddrInst::Adda | RegAddrInst::Suba |
            RegAddrInst::Addl | RegAddrInst::Subl | RegAddrInst::Mula |
            RegAddrInst::Diva | RegAddrInst::Mull | RegAddrInst::Divl |
            RegAddrInst::And | RegAddrInst::Or | RegAddrInst::Xor |
            RegAddrInst::Cpa | RegAddrInst::Cpl
        )
    }
}

impl TwoRegInst {
    pub fn from_reg_addr(inst: RegAddrInst) -> Option<Self> {
        match inst {
            RegAddrInst::Ld => Some(TwoRegInst::Ld),
            RegAddrInst::Adda => Some(TwoRegInst::Adda),
            RegAddrInst::Suba => Some(TwoRegInst::Suba),
            RegAddrInst::Addl => Some(TwoRegInst::Addl),
            RegAddrInst::Subl => Some(TwoRegInst::Subl),
            RegAddrInst::Mula => Some(TwoRegInst::Mula),
            RegAddrInst::Diva => Some(TwoRegInst::Diva),
            RegAddrInst::Mull => Some(TwoRegInst::Mull),
            RegAddrInst::Divl => Some(TwoRegInst::Divl),
            RegAddrInst::And => Some(TwoRegInst::And),
            RegAddrInst::Or => Some(TwoRegInst::Or),
            RegAddrInst::Xor => Some(TwoRegInst::Xor),
            RegAddrInst::Cpa => Some(TwoRegInst::Cpa),
            RegAddrInst::Cpl => Some(TwoRegInst::Cpl),
            _ => None,
        }
    }
}

impl AddrInst {
    pub fn from_str(s: &str) -> Option<Self> {
        match s {
            "JMI" => Some(AddrInst::Jmi),
            "JNZ" => Some(AddrInst::Jnz),
            "JZE" => Some(AddrInst::Jze),
            "JUMP" => Some(AddrInst::Jump),
            "JPL" => Some(AddrInst::Jpl),
            "JOV" => Some(AddrInst::Jov),
            "PUSH" => Some(AddrInst::Push),
            "CALL" => Some(AddrInst::Call),
            "SVC" => Some(AddrInst::Svc),
            _ => None,
        }
    }
}
