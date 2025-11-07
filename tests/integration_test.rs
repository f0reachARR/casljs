use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

use c2c2::lexer::Lexer;
use c2c2::parser::Parser;
use c2c2::assembler::Assembler;
use c2c2::emulator::Comet2;

fn parse_input_json(content: &str) -> HashMap<String, Vec<String>> {
    let mut result = HashMap::new();
    let mut current_key = String::new();
    let mut current_values = Vec::new();
    let mut in_array = false;
    let mut in_string = false;
    let mut current_string = String::new();

    let mut chars = content.chars().peekable();

    while let Some(ch) = chars.next() {
        match ch {
            '"' => {
                if in_string {
                    // End of string
                    if in_array {
                        current_values.push(current_string.clone());
                        current_string.clear();
                    } else {
                        current_key = current_string.clone();
                        current_string.clear();
                    }
                    in_string = false;
                } else {
                    in_string = true;
                }
            }
            '[' => {
                in_array = true;
                current_values.clear();
            }
            ']' => {
                in_array = false;
                if !current_key.is_empty() {
                    result.insert(current_key.clone(), current_values.clone());
                    current_key.clear();
                }
            }
            _ => {
                if in_string {
                    current_string.push(ch);
                }
            }
        }
    }

    result
}

fn load_input_json() -> HashMap<String, Vec<String>> {
    let input_path = Path::new("test/input.json");
    let content = fs::read_to_string(input_path)
        .expect("Failed to read input.json");
    parse_input_json(&content)
}

fn find_cas_files() -> Vec<PathBuf> {
    let mut files = Vec::new();
    let samples_dir = Path::new("test/samples");

    if let Ok(entries) = fs::read_dir(samples_dir) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                // Recursively search in subdirectories
                if let Ok(sub_entries) = fs::read_dir(&path) {
                    for sub_entry in sub_entries.flatten() {
                        let sub_path = sub_entry.path();
                        if sub_path.extension().and_then(|s| s.to_str()) == Some("cas") {
                            files.push(sub_path);
                        }
                    }
                }
            }
        }
    }

    files.sort();
    files
}

fn run_test_program(cas_file: &Path, inputs: &[String]) -> Result<String, String> {
    // Read source file
    let source = fs::read_to_string(cas_file)
        .map_err(|e| format!("Failed to read file: {}", e))?;

    // Assemble
    let mut lexer = Lexer::new(&source);
    let tokens = lexer.tokenize()
        .map_err(|e| format!("Lexer error: {}", e))?;

    let mut parser = Parser::new(tokens);
    let ast = parser.parse()
        .map_err(|e| format!("Parser error: {}", e))?;

    let assembler = Assembler::new(ast);
    let (binary, start_address, _) = assembler.assemble()
        .map_err(|e| format!("Assembler error: {}", e))?;

    // Run emulator
    let mut emulator = Comet2::new(&binary, start_address, inputs, false); // quiet=false to capture output
    emulator.run()
        .map_err(|e| format!("Runtime error: {}", e))?;

    Ok(emulator.get_output())
}

#[test]
fn test_all_samples() {
    let input_map = load_input_json();
    let cas_files = find_cas_files();

    let mut passed = 0;
    let mut failed = 0;
    let mut failures = Vec::new();

    for cas_file in &cas_files {
        let filename = cas_file.file_name().unwrap().to_str().unwrap();
        let expect_file = PathBuf::from(format!("test/test_expects/{}.out", filename));

        // Skip if no expected output file exists
        if !expect_file.exists() {
            eprintln!("Skipping {} (no expected output)", filename);
            continue;
        }

        // Get inputs for this test
        let inputs = input_map.get(filename)
            .map(|v| v.as_slice())
            .unwrap_or(&[]);

        // Run the test
        match run_test_program(&cas_file, inputs) {
            Ok(output) => {
                // Read expected output
                match fs::read_to_string(&expect_file) {
                    Ok(expected) => {
                        if output == expected {
                            passed += 1;
                            eprintln!("✓ {}", filename);
                        } else {
                            failed += 1;
                            eprintln!("✗ {} - output mismatch", filename);
                            failures.push(format!(
                                "\n{}\nExpected:\n{}\nActual:\n{}\n",
                                filename, expected, output
                            ));
                        }
                    }
                    Err(e) => {
                        failed += 1;
                        eprintln!("✗ {} - failed to read expected output: {}", filename, e);
                        failures.push(format!("{}: failed to read expected output", filename));
                    }
                }
            }
            Err(e) => {
                failed += 1;
                eprintln!("✗ {} - {}", filename, e);
                failures.push(format!("{}: {}", filename, e));
            }
        }
    }

    eprintln!("\n{} passed, {} failed", passed, failed);

    if !failures.is_empty() {
        eprintln!("\nFailures:");
        for failure in &failures {
            eprintln!("{}", failure);
        }
        panic!("Some tests failed");
    }
}
