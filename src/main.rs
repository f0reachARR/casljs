// c2c2 - CASL II Assembler / COMET II Emulator
// Rust implementation
// Version 1.0.4 KIT (Jan 23, 2025)

mod lexer;
mod ast;
mod parser;

use std::collections::HashMap;
use std::env;
use std::fs;
use std::io::{self, Write};
use std::process;

use lexer::Lexer;
use parser::Parser;

const VERSION: &str = "1.0.4 KIT (Jan 23, 2025)";

// System call addresses
const SYS_IN: u16 = 0xfff0;
const SYS_OUT: u16 = 0xfff2;
const EXIT_USR: u16 = 0x0000;

// Flag register bits
const FR_PLUS: u16 = 0;
const FR_ZERO: u16 = 1;
const FR_MINUS: u16 = 2;
const FR_OVER: u16 = 4;

// Stack
const STACK_TOP: u16 = 0xff00;

// Register indices
const PC: usize = 0;
const FR: usize = 1;
const GR0: usize = 2;
const SP: usize = 10;

type Result<T> = std::result::Result<T, String>;

#[derive(Debug, Clone)]
struct Options {
    opt_a: bool,  // show detailed info
    opt_c: bool,  // casl2 only
    opt_r: bool,  // run immediately
    opt_n: bool,  // no color
    opt_q: bool,  // quiet
    opt_Q: bool,  // very quiet
}

impl Default for Options {
    fn default() -> Self {
        Self {
            opt_a: false,
            opt_c: false,
            opt_r: false,
            opt_n: false,
            opt_q: false,
            opt_Q: false,
        }
    }
}

struct Args {
    casl2file: String,
    inputs: Vec<String>,
    options: Options,
}

fn parse_args() -> Result<Args> {
    let args: Vec<String> = env::args().collect();

    if args.len() < 2 {
        return Err("Usage: c2c2 [options] <casl2file> [input1 ...]".to_string());
    }

    let mut options = Options::default();
    let mut casl2file = String::new();
    let mut inputs = Vec::new();
    let mut i = 1;

    while i < args.len() {
        match args[i].as_str() {
            "-a" | "--all" => options.opt_a = true,
            "-c" | "--casl" => options.opt_c = true,
            "-r" | "--run" => options.opt_r = true,
            "-n" | "--nocolor" => options.opt_n = true,
            "-q" | "--quiet" => options.opt_q = true,
            "-Q" | "--QuietRun" => {
                options.opt_Q = true;
                options.opt_q = true;
                options.opt_r = true;
            }
            "-h" | "--help" => {
                print_usage();
                process::exit(0);
            }
            "-V" | "--version" => {
                println!("c2c2 version {}", VERSION);
                process::exit(0);
            }
            arg => {
                if arg.starts_with('-') {
                    return Err(format!("Unknown option: {}", arg));
                }
                if casl2file.is_empty() {
                    casl2file = arg.to_string();
                } else {
                    inputs.push(arg.to_string());
                }
            }
        }
        i += 1;
    }

    if casl2file.is_empty() {
        return Err("No CASL2 source file specified".to_string());
    }

    Ok(Args {
        casl2file,
        inputs,
        options,
    })
}

fn print_usage() {
    println!("Usage: c2c2 [options] <casl2file> [input1 ...]");
    println!();
    println!("Options:");
    println!("  -V, --version   output the version number");
    println!("  -a, --all       [casl2] show detailed info");
    println!("  -c, --casl      [casl2] apply casl2 only");
    println!("  -r, --run       [comet2] run immediately");
    println!("  -n, --nocolor   [casl2/comet2] disable color messages");
    println!("  -q, --quiet     [casl2/comet2] be quiet");
    println!("  -Q, --QuietRun  [comet2] be QUIET! (implies -q and -r)");
    println!("  -h, --help      display help for command");
}

fn main() {
    let args = match parse_args() {
        Ok(args) => args,
        Err(e) => {
            eprintln!("{}", e);
            process::exit(1);
        }
    };

    if let Err(e) = run(args) {
        eprintln!("{}", e);
        process::exit(1);
    }
}

fn run(args: Args) -> Result<()> {
    let options = &args.options;

    // Print CASL2 banner
    if !options.opt_q {
        println!("   _________   _____ __       ________");
        println!("  / ____/   | / ___// /      /  _/  _/");
        println!(" / /   / /| | \\__ \\/ /       / / / /");
        println!("/ /___/ ___ |___/ / /___   _/ /_/ /");
        println!("\\____/_/  |_/____/_____/  /___/___/   ");
        println!("This is CASL II, version {}.", VERSION);
        println!("(c) 2001-2023, Osamu Mizuno.\n");
    }

    // Read source file
    let source = fs::read_to_string(&args.casl2file)
        .map_err(|e| format!("Failed to read file {}: {}", args.casl2file, e))?;

    // Assemble
    let (comet2bin, start_address, address_max) = assemble(&source, options)?;

    if !options.opt_q {
        println!("Successfully assembled.");
    }

    if options.opt_c {
        return Ok(());
    }

    // Print COMET2 banner
    if !options.opt_q {
        println!("   __________  __  _______________   ________");
        println!("  / ____/ __ \\/  |/  / ____/_  __/  /  _/  _/");
        println!(" / /   / / / / /|_/ / __/   / /     / / / /");
        println!("/ /___/ /_/ / /  / / /___  / /    _/ /_/ /");
        println!("\\____/\\____/_/  /_/_____/ /_/    /___/___/  ");
        println!("This is COMET II, version {}.", VERSION);
        println!("(c) 2001-2023, Osamu Mizuno.\n");
        println!("Loading comet2 binary ... done");
    }

    // Run emulator
    if options.opt_r {
        execute(&comet2bin, start_address, address_max, &args.inputs, options)?;
    }

    Ok(())
}

fn assemble(source: &str, options: &Options) -> Result<(Vec<u16>, u16, u16)> {
    // Lexical analysis
    let mut lexer = Lexer::new(source);
    let tokens = lexer.tokenize()
        .map_err(|e| format!("Lexer error: {}", e))?;

    if !options.opt_q && options.opt_a {
        println!("=== Tokens ===");
        for (token, line) in &tokens {
            println!("Line {}: {:?}", line, token);
        }
        println!();
    }

    // Syntax analysis
    let mut parser = Parser::new(tokens);
    let program = parser.parse()
        .map_err(|e| format!("Parser error: {}", e))?;

    if !options.opt_q && options.opt_a {
        println!("=== AST ===");
        for line in &program.lines {
            println!("Line {}: {:?}", line.line_number, line);
        }
        println!();
    }

    // Code generation (stub)
    // TODO: Implement full code generation
    Err("Code generation not yet fully implemented - work in progress".to_string())
}

fn execute(
    memory: &[u16],
    start_address: u16,
    address_max: u16,
    inputs: &[String],
    options: &Options,
) -> Result<()> {
    // Placeholder: This is a minimal stub
    // Full implementation would include:
    // - Instruction decode and execute loop
    // - Register state management
    // - I/O operations

    Err("Execution not yet fully implemented - work in progress".to_string())
}
