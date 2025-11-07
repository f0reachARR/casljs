package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func assemble(inputFilepath string, asmState *AssemblerState) ([]uint16, string, error) {
	// Read source file
	content, err := ioutil.ReadFile(inputFilepath)
	if err != nil {
		return nil, "", fmt.Errorf("[CASL2 ERROR] Cannot read file: %v", err)
	}

	casl2code := string(content)
	asmState.file = inputFilepath

	// Pass 1: Build symbol table
	startLabel, err := pass1(casl2code, asmState)
	if err != nil {
		return nil, "", err
	}

	// Pass 2: Generate binary
	comet2bin, err := pass2(asmState)
	if err != nil {
		return nil, "", err
	}

	return comet2bin, startLabel, nil
}

func pass1(source string, asmState *AssemblerState) (string, error) {
	var inBlock bool
	var address int
	var literalStack []string
	var comet2startLabel string

	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	asmState.line = 0

	for i, line := range lines {
		asmState.line = i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse the line using the new lexer-based parser
		parsed, err := ParseLine(line, asmState.line)
		if err != nil {
			return "", errorCasl2(asmState, fmt.Sprintf("Syntax error: %s", err))
		}

		// Extract label, instruction, and operands from parsed result
		label := parsed.Label
		inst := parsed.Instruction
		var opr string
		if len(parsed.Operands) > 0 {
			opr = strings.Join(parsed.Operands, ",")
		}

		// Keep every line in buf
		uniqLabel := ""
		if label != "" {
			uniqLabel = asmState.varScope + ":" + label
		}
		asmState.buf = append(asmState.buf, uniqLabel+"\t"+inst+"\t"+opr)

		// Register label to symbol table
		if label != "" && inBlock {
			err := addLabel(asmState, label, address)
			if err != nil {
				return "", err
			}

			// Check if label is referred from START instruction
			if label == asmState.actualLabel {
				err := updateLabel(asmState, asmState.virtualLabel, address)
				if err != nil {
					return "", err
				}
				asmState.actualLabel = ""
			}
		}

		// Generate object code according to instruction type
		if inst != "" {
			instDef, ok := CASL2TBL[inst]
			if !ok {
				return "", errorCasl2(asmState, fmt.Sprintf("Illegal instruction \"%s\"", inst))
			}

			instType := instDef.Type

			// Parse operands - already parsed by ParseLine
			oprArray := parsed.Operands

			// START must be the first instruction
			if !inBlock && instType != START {
				return "", errorCasl2(asmState, "NO \"START\" instruction found")
			}

			// GR0 cannot be used as index register
			if len(oprArray) > 2 {
				if isGR0(oprArray[2]) {
					return "", errorCasl2(asmState, "Can't use GR0 as an index register")
				}
			}

			// Process each instruction type
			switch instType {
			case OP1:
				if len(oprArray) < 2 || len(oprArray) > 3 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				if len(oprArray) == 2 {
					oprArray = append(oprArray, "0")
				}

				// Handle literals
				if strings.HasPrefix(oprArray[1], "=") {
					oprArray[1] = handleLiteral(oprArray[1], &literalStack, &asmState.literalCounter)
				} else if IsValidLabel(oprArray[1]) && !IsRegister(oprArray[1]) {
					oprArray[1] = asmState.varScope + ":" + oprArray[1]
				}

				genCode2(asmState.memory, address, int(instDef.Code), oprArray[0], oprArray[1], oprArray[2], asmState)
				address += 2

			case OP2:
				if len(oprArray) < 1 || len(oprArray) > 2 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				if len(oprArray) == 1 {
					oprArray = append(oprArray, "0")
				}

				if !IsRegister(oprArray[0]) && IsValidLabel(oprArray[0]) {
					if strings.Contains(inst, "CALL") {
						oprArray[0] = "CALL_" + asmState.varScope + ":" + oprArray[0]
					} else {
						oprArray[0] = asmState.varScope + ":" + oprArray[0]
					}
				}

				genCode2(asmState.memory, address, int(instDef.Code), "0", oprArray[0], oprArray[1], asmState)
				address += 2

			case OP3:
				if len(oprArray) != 1 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				genCode3(asmState.memory, address, int(instDef.Code), oprArray[0], "0", asmState)
				address++

			case OP4:
				if len(oprArray) != 0 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				genCode1(asmState.memory, address, int(instDef.Code)<<8, asmState)
				address++

			case OP5:
				if len(oprArray) < 2 || len(oprArray) > 3 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				if len(oprArray) == 2 {
					oprArray = append(oprArray, "0")
				}

				// Handle literals
				if strings.HasPrefix(oprArray[1], "=") {
					oprArray[1] = handleLiteral(oprArray[1], &literalStack, &asmState.literalCounter)
				} else if IsValidLabel(oprArray[1]) && !IsRegister(oprArray[1]) {
					oprArray[1] = asmState.varScope + ":" + oprArray[1]
				}

				// Check if GR,GR form
				if IsRegister(oprArray[1]) {
					instCode := int(instDef.Code) + 4
					genCode3(asmState.memory, address, instCode, oprArray[0], oprArray[1], asmState)
					address++
				} else {
					genCode2(asmState.memory, address, int(instDef.Code), oprArray[0], oprArray[1], oprArray[2], asmState)
					address += 2
				}

			case START:
				if label == "" {
					return "", errorCasl2(asmState, "No label found at START")
				}

				if asmState.firstStart {
					asmState.firstStart = false
					if len(oprArray) > 0 {
						comet2startLabel = label + ":" + oprArray[0]
					} else {
						comet2startLabel = label + ":" + label
					}
				} else {
					if len(oprArray) > 0 {
						asmState.actualLabel = oprArray[0]
					} else {
						asmState.actualLabel = ""
					}
					asmState.virtualLabel = label
				}

				asmState.varScope = label
				err := addLabel(asmState, label, address)
				if err != nil {
					return "", err
				}
				inBlock = true

			case END:
				if label != "" {
					return "", errorCasl2(asmState, fmt.Sprintf("Can't use label \"%s\" at END", label))
				}
				if len(oprArray) != 0 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}

				// Expand literals
				for _, lit := range literalStack {
					addLiteral(asmState, lit, address)
					lit = strings.TrimPrefix(lit, "=")

					if strings.HasPrefix(lit, "'") && strings.HasSuffix(lit, "'") {
						str := lit[1 : len(lit)-1]
						str = strings.ReplaceAll(str, "''", "'")
						for _, ch := range str {
							genCode1(asmState.memory, address, int(ch), asmState)
							address++
						}
						genCode1(asmState.memory, address, 0, asmState)
						address++
					} else if isNumberOrHex(lit) {
						genCode1(asmState.memory, address, lit, asmState)
						address++
					} else {
						return "", errorCasl2(asmState, fmt.Sprintf("Invalid literal =%s", lit))
					}
				}

				asmState.varScope = ""
				inBlock = false

			case DS:
				if len(oprArray) != 1 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				count, err := strconv.Atoi(oprArray[0])
				if err != nil {
					return "", errorCasl2(asmState, fmt.Sprintf("\"%s\" must be decimal", oprArray[0]))
				}
				for j := 0; j < count; j++ {
					genCode1(asmState.memory, address, 0, asmState)
					address++
				}

			case DC:
				if len(oprArray) < 1 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				for _, op := range oprArray {
					if strings.HasPrefix(op, "'") && strings.HasSuffix(op, "'") {
						str := op[1 : len(op)-1]
						str = strings.ReplaceAll(str, "''", "'")
						for _, ch := range str {
							genCode1(asmState.memory, address, int(ch), asmState)
							address++
						}
						genCode1(asmState.memory, address, 0, asmState)
						address++
					} else if IsValidLabel(op) {
						op = asmState.varScope + ":" + op
						genCode1(asmState.memory, address, op, asmState)
						address++
					} else {
						genCode1(asmState.memory, address, op, asmState)
						address++
					}
				}

			case IN, OUT:
				if len(oprArray) != 2 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}

				checkLabel(asmState, oprArray[0])
				checkLabel(asmState, oprArray[1])

				oprArray[0] = asmState.varScope + ":" + oprArray[0]
				oprArray[1] = asmState.varScope + ":" + oprArray[1]

				entry := SYS_IN
				if instType == OUT {
					entry = SYS_OUT
				}

				genCode2(asmState.memory, address, int(CASL2TBL["PUSH"].Code), "0", "0", "1", asmState)
				genCode2(asmState.memory, address+2, int(CASL2TBL["PUSH"].Code), "0", "0", "2", asmState)
				genCode2(asmState.memory, address+4, int(CASL2TBL["LAD"].Code), "1", oprArray[0], "0", asmState)
				genCode2(asmState.memory, address+6, int(CASL2TBL["LAD"].Code), "2", oprArray[1], "0", asmState)
				genCode2(asmState.memory, address+8, int(CASL2TBL["SVC"].Code), "0", strconv.Itoa(entry), "0", asmState)
				genCode3(asmState.memory, address+10, int(CASL2TBL["POP"].Code), "2", "0", asmState)
				genCode3(asmState.memory, address+11, int(CASL2TBL["POP"].Code), "1", "0", asmState)
				address += 12

			case RPUSH:
				if len(oprArray) != 0 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				for j := 0; j < 7; j++ {
					genCode2(asmState.memory, address+j*2, int(CASL2TBL["PUSH"].Code), "0", "0", strconv.Itoa(j+1), asmState)
				}
				address += 14

			case RPOP:
				if len(oprArray) != 0 {
					return "", errorCasl2(asmState, fmt.Sprintf("Invalid operand \"%s\"", opr))
				}
				for j := 0; j < 7; j++ {
					genCode3(asmState.memory, address+j, int(CASL2TBL["POP"].Code), strconv.Itoa(7-j), "0", asmState)
				}
				address += 7

			default:
				return "", errorCasl2(asmState, fmt.Sprintf("Instruction type \"%s\" is not implemented", instType))
			}
		}
	}

	if inBlock {
		return "", errorCasl2(asmState, "NO \"END\" instruction found")
	}

	addressMax = address
	return comet2startLabel, nil
}

func pass2(asmState *AssemblerState) ([]uint16, error) {
	if *optAll {
		caslPrint("CASL LISTING\n")
	}

	var lastLine = -1

	// Sort memory addresses
	var addresses []int
	for addr := range asmState.memory {
		if addr >= 0 {
			addresses = append(addresses, addr)
		}
	}

	// Simple sort
	for i := 0; i < len(addresses); i++ {
		for j := i + 1; j < len(addresses); j++ {
			if addresses[i] > addresses[j] {
				addresses[i], addresses[j] = addresses[j], addresses[i]
			}
		}
	}

	comet2bin := make([]uint16, 0)
	for _, address := range addresses {
		memEntry := asmState.memory[address]
		asmState.line = memEntry.Line

		val := expandLabel(asmState.symtbl, memEntry.Val)
		comet2bin = append(comet2bin, uint16(val))

		if *optAll {
			bufLine := strings.Split(asmState.buf[asmState.line-1], "\t")
			if len(bufLine) > 0 {
				// Extract label after colon
				if idx := strings.LastIndex(bufLine[0], ":"); idx >= 0 {
					bufLine[0] = bufLine[0][idx+1:]
				}
			}
			line := strings.Join(bufLine, "\t")

			if asmState.line != lastLine {
				str := fmt.Sprintf("%4d %s %s\t%s", asmState.line, hex(address, 4), hex(val, 4), line)
				asmState.outdump = append(asmState.outdump, str)
				lastLine = asmState.line
			} else {
				str := fmt.Sprintf("%4d      %s", asmState.line, hex(val, 4))
				asmState.outdump = append(asmState.outdump, str)
			}
		}
	}

	if *optAll {
		asmState.outdump = append(asmState.outdump, "\nDEFINED SYMBOLS")

		// Sort symbols by line
		type symInfo struct {
			name string
			line int
		}
		var symbols []symInfo
		for name, entry := range asmState.symtbl {
			if !strings.HasPrefix(name, "=") {
				symbols = append(symbols, symInfo{name, entry.Line})
			}
		}

		// Sort by line
		for i := 0; i < len(symbols); i++ {
			for j := i + 1; j < len(symbols); j++ {
				if symbols[i].line > symbols[j].line {
					symbols[i], symbols[j] = symbols[j], symbols[i]
				}
			}
		}

		for _, sym := range symbols {
			label := sym.name
			// Parse scope:label format
			parts := strings.Split(label, ":")
			if len(parts) == 2 {
				var labelView string
				if parts[0] == parts[1] {
					labelView = parts[1]
				} else {
					labelView = fmt.Sprintf("%s (%s)", parts[1], parts[0])
				}
				val := expandLabel(asmState.symtbl, label)
				asmState.outdump = append(asmState.outdump, fmt.Sprintf("%d:\t%s\t%s", sym.line, hex(val, 4), labelView))
			}
		}

		for _, line := range asmState.outdump {
			caslPrint(line)
		}
	}

	return comet2bin, nil
}

// Helper functions

func parseOperands(opr string) []string {
	var result []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(opr); i++ {
		ch := opr[i]

		if ch == '\'' {
			if inQuote && i+1 < len(opr) && opr[i+1] == '\'' {
				current.WriteByte(ch)
				current.WriteByte(ch)
				i++
			} else {
				inQuote = !inQuote
				current.WriteByte(ch)
			}
		} else if ch == ',' && !inQuote {
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

func handleLiteral(lit string, stack *[]string, counter *int) string {
	newLit := fmt.Sprintf("%s_%d", lit, *counter)
	*stack = append(*stack, newLit)
	*counter++
	return newLit
}

func checkLabel(asmState *AssemblerState, label string) error {
	if !IsValidLabel(label) {
		return errorCasl2(asmState, fmt.Sprintf("Invalid label \"%s\"", label))
	}
	return nil
}

func addLabel(asmState *AssemblerState, label string, val int) error {
	if err := checkLabel(asmState, label); err != nil {
		return err
	}

	uniqLabel := asmState.varScope + ":" + label
	if _, exists := asmState.symtbl[uniqLabel]; exists {
		return errorCasl2(asmState, fmt.Sprintf("Label \"%s\" has already defined", label))
	}

	asmState.symtbl[uniqLabel] = &SymbolEntry{
		Val:  val,
		File: asmState.file,
		Line: asmState.line,
	}

	return nil
}

func updateLabel(asmState *AssemblerState, label string, val int) error {
	if err := checkLabel(asmState, label); err != nil {
		return err
	}

	uniqLabel := asmState.varScope + ":" + label
	if _, exists := asmState.symtbl[uniqLabel]; !exists {
		return errorCasl2(asmState, fmt.Sprintf("Label \"%s\" is not defined", label))
	}

	asmState.symtbl[uniqLabel] = &SymbolEntry{
		Val:  val,
		File: asmState.file,
		Line: asmState.line,
	}

	return nil
}

func addLiteral(asmState *AssemblerState, literal string, val int) {
	asmState.symtbl[literal] = &SymbolEntry{
		Val:  val,
		File: asmState.file,
		Line: asmState.line,
	}
}

func expandLabel(symtbl map[string]*SymbolEntry, val interface{}) int {
	switch v := val.(type) {
	case int:
		return v & 0xffff
	case string:
		// Check if it's a hex number
		if strings.HasPrefix(v, "#") {
			num, err := strconv.ParseInt(v[1:], 16, 64)
			if err == nil {
				// Safe: masked to 16 bits
				return int(num & 0xffff)
			}
		}

		// Check if it's in symbol table
		if entry, exists := symtbl[v]; exists {
			return expandLabel(symtbl, entry.Val)
		}

		// Check for CALL_ prefix
		if strings.HasPrefix(v, "CALL_") {
			lbl := v[5:]
			if entry, exists := symtbl[lbl]; exists {
				return expandLabel(symtbl, entry.Val)
			}

			// Try with scope - extract label after colon
			if idx := strings.LastIndex(v, ":"); idx >= 0 {
				labelPart := v[idx+1:]
				k := labelPart + ":" + labelPart
				if entry, exists := symtbl[k]; exists {
					return expandLabel(symtbl, entry.Val)
				}
			}
		}

		// Try to parse as decimal
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			// Safe: masked to 16 bits
			return int(num & 0xffff)
		}

		// If all else fails, return 0
		return 0
	default:
		return 0
	}
}

func checkRegister(register string) (int, error) {
	// Use the lexer's CheckRegister function
	return CheckRegister(register)
}

// isNumberOrHex checks if a string is a number or hex number without regex
func isNumberOrHex(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// Check for hex
	if s[0] == '#' {
		if len(s) == 1 {
			return false
		}
		for i := 1; i < len(s); i++ {
			ch := s[i]
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
		return true
	}
	
	// Check for signed decimal
	start := 0
	if s[0] == '+' || s[0] == '-' {
		start = 1
	}
	
	if start >= len(s) {
		return false
	}
	
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isGR0 checks if an operand is GR0 without regex
func isGR0(s string) bool {
	s = strings.ToUpper(s)
	return s == "GR0" || s == "0"
}

func genCode1(memory map[int]*MemoryEntry, address int, val interface{}, asmState *AssemblerState) {
	switch v := val.(type) {
	case int:
		memory[address] = &MemoryEntry{Val: v, File: asmState.file, Line: asmState.line}
	case string:
		// Check for hex
		if strings.HasPrefix(v, "#") {
			if num, err := strconv.ParseInt(v[1:], 16, 64); err == nil {
				// Safe: COMET2 uses 16-bit values
				memory[address] = &MemoryEntry{Val: int(num & 0xffff), File: asmState.file, Line: asmState.line}
				return
			}
		}
		// Check for decimal
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			// Safe: COMET2 uses 16-bit values
			memory[address] = &MemoryEntry{Val: int(num & 0xffff), File: asmState.file, Line: asmState.line}
			return
		}
		// Store as string (will be resolved in pass2)
		memory[address] = &MemoryEntry{Val: v, File: asmState.file, Line: asmState.line}
	}
}

func genCode2(memory map[int]*MemoryEntry, address int, code int, gr, adr, xr string, asmState *AssemblerState) {
	ngr, _ := checkRegister(gr)
	nxr, _ := checkRegister(xr)

	val := (code << 8) + (ngr << 4) + nxr
	memory[address] = &MemoryEntry{Val: val, File: asmState.file, Line: asmState.line}

	// Handle address operand
	if strings.HasPrefix(adr, "#") {
		if num, err := strconv.ParseInt(adr[1:], 16, 64); err == nil {
			// Safe: COMET2 uses 16-bit addresses
			memory[address+1] = &MemoryEntry{Val: int(num & 0xffff), File: asmState.file, Line: asmState.line}
			return
		}
	}

	memory[address+1] = &MemoryEntry{Val: adr, File: asmState.file, Line: asmState.line}
}

func genCode3(memory map[int]*MemoryEntry, address int, code int, gr1, gr2 string, asmState *AssemblerState) {
	ngr1, _ := checkRegister(gr1)
	ngr2, _ := checkRegister(gr2)

	val := (code << 8) + (ngr1 << 4) + ngr2
	memory[address] = &MemoryEntry{Val: val, File: asmState.file, Line: asmState.line}
}

func errorCasl2(asmState *AssemblerState, msg string) error {
	return fmt.Errorf("%sLine %d: %s%s",
		"\x1b[31;43m", asmState.line, msg, "\x1b[0m")
}
