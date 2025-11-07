package main

import (
	"fmt"
	"regexp"
	"strings"
)

// COMET2 instruction table
type Comet2Instruction struct {
	ID   string
	Type InstructionType
}

var COMET2TBL = map[int]Comet2Instruction{
	0x00: {"NOP", OP4},
	0x10: {"LD", OP1},
	0x11: {"ST", OP1},
	0x12: {"LAD", OP1},
	0x14: {"LD", OP5},
	0x20: {"ADDA", OP1},
	0x21: {"SUBA", OP1},
	0x22: {"ADDL", OP1},
	0x23: {"SUBL", OP1},
	0x24: {"ADDA", OP5},
	0x25: {"SUBA", OP5},
	0x26: {"ADDL", OP5},
	0x27: {"SUBL", OP5},
	0x28: {"MULA", OP1},
	0x29: {"DIVA", OP1},
	0x2a: {"MULL", OP1},
	0x2b: {"DIVL", OP1},
	0x2c: {"MULA", OP5},
	0x2d: {"DIVA", OP5},
	0x2e: {"MULL", OP5},
	0x2f: {"DIVL", OP5},
	0x30: {"AND", OP1},
	0x31: {"OR", OP1},
	0x32: {"XOR", OP1},
	0x34: {"AND", OP5},
	0x35: {"OR", OP5},
	0x36: {"XOR", OP5},
	0x40: {"CPA", OP1},
	0x41: {"CPL", OP1},
	0x44: {"CPA", OP5},
	0x45: {"CPL", OP5},
	0x50: {"SLA", OP1},
	0x51: {"SRA", OP1},
	0x52: {"SLL", OP1},
	0x53: {"SRL", OP1},
	0x61: {"JMI", OP2},
	0x62: {"JNZ", OP2},
	0x63: {"JZE", OP2},
	0x64: {"JUMP", OP2},
	0x65: {"JPL", OP2},
	0x66: {"JOV", OP2},
	0x70: {"PUSH", OP2},
	0x71: {"POP", OP3},
	0x80: {"CALL", OP2},
	0x81: {"RET", OP4},
	0xf0: {"SVC", OP2},
}

func parse(memory []uint16, state []int) (string, string, int) {
	pc := state[PC]
	inst := memGet(memory, pc) >> 8
	gr := (memGet(memory, pc) >> 4) & 0xf
	xr := memGet(memory, pc) & 0xf
	adr := memGet(memory, pc+1)

	instSym := "DC"
	oprSym := fmt.Sprintf("#%s", hex(memGet(memory, pc), 4))
	size := 1

	if comet2Inst, ok := COMET2TBL[inst]; ok {
		instSym = comet2Inst.ID
		instType := comet2Inst.Type

		switch instType {
		case OP1:
			oprSym = fmt.Sprintf("GR%d,   #%s", gr, hex(adr, 4))
			if xr > 0 {
				oprSym += fmt.Sprintf(", GR%d", xr)
			}
			size = 2
		case OP2:
			oprSym = fmt.Sprintf("#%s", hex(adr, 4))
			if xr > 0 {
				oprSym += fmt.Sprintf(", GR%d", xr)
			}
			size = 2
		case OP3:
			oprSym = fmt.Sprintf("GR%d", gr)
			size = 1
		case OP4:
			oprSym = ""
			size = 1
		case OP5:
			oprSym = fmt.Sprintf("GR%d, GR%d", gr, xr)
			size = 1
		}
	}

	return instSym, oprSym, size
}

func execIn(memory []uint16, state []int, text string) {
	text = strings.TrimSpace(text)
	if len(text) > 256 {
		text = text[:256]
	}

	lenp := state[GR2]
	bufp := state[GR1]

	memPut(memory, lenp, len(text))
	for i, ch := range text {
		memPut(memory, bufp+i, int(ch))
	}

	state[PC] += 2
}

func execOut(memory []uint16, state []int) {
	lenp := state[GR2]
	bufp := state[GR1]
	length := memGet(memory, lenp)

	var outstr strings.Builder
	for i := 0; i < length; i++ {
		outstr.WriteByte(byte(memGet(memory, bufp+i) & 0xff))
	}

	cometOut(outstr.String())
}

func stepExec(memory []uint16, state []int) (bool, error) {
	inst, opr, _ := parse(memory, state)

	pc := state[PC]
	fr := state[FR]
	sp := state[SP]
	regs := state[GR0 : GR7+1]

	instVal := memGet(memory, pc)
	gr := (instVal >> 4) & 0xf
	xr := instVal & 0xf
	adr := memGet(memory, pc+1)
	eadr := adr

	var val int
	stopFlag := false

	if xr >= 1 && xr <= 7 {
		eadr += regs[xr]
	}
	eadr &= 0xffff

	grIsGr := regexp.MustCompile(`GR[0-7], GR[0-7]`)

	switch inst {
	case "LD":
		if !grIsGr.MatchString(opr) {
			regs[gr] = memGet(memory, eadr)
			fr = getFlag(regs[gr])
			pc += 2
		} else {
			regs[gr] = regs[xr]
			fr = getFlag(regs[gr])
			pc++
		}

	case "ST":
		memPut(memory, eadr, regs[gr])
		pc += 2

	case "LAD":
		regs[gr] = eadr
		pc += 2

	case "ADDA":
		if !grIsGr.MatchString(opr) {
			regs[gr] = signed(regs[gr])
			regs[gr] += memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] = signed(regs[gr])
			regs[xr] = signed(regs[xr])
			regs[gr] += regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			regs[xr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "SUBA":
		if !grIsGr.MatchString(opr) {
			regs[gr] = signed(regs[gr])
			regs[gr] -= memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] = signed(regs[gr])
			regs[xr] = signed(regs[xr])
			regs[gr] -= regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			regs[xr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "ADDL":
		if !grIsGr.MatchString(opr) {
			regs[gr] += memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] += regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "SUBL":
		if !grIsGr.MatchString(opr) {
			regs[gr] -= memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] -= regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "MULA":
		if !grIsGr.MatchString(opr) {
			regs[gr] = signed(regs[gr])
			regs[gr] *= memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] = signed(regs[gr])
			regs[xr] = signed(regs[xr])
			regs[gr] *= regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > MAX_SIGNED {
				ofr1 = FR_OVER
			}
			if regs[gr] < MIN_SIGNED {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			regs[xr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "MULL":
		if !grIsGr.MatchString(opr) {
			regs[gr] *= memGet(memory, eadr)
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc += 2
		} else {
			regs[gr] *= regs[xr]
			ofr1 := 0
			ofr2 := 0
			if regs[gr] > 0xffff {
				ofr1 = FR_OVER
			}
			if regs[gr] < 0 {
				ofr2 = FR_OVER
			}
			regs[gr] &= 0xffff
			regs[xr] &= 0xffff
			fr = getFlag(regs[gr]) | ofr1 | ofr2
			pc++
		}

	case "DIVA":
		if !grIsGr.MatchString(opr) {
			regs[gr] = signed(regs[gr])
			m := memGet(memory, eadr)
			if m == 0 {
				fr = FR_OVER | FR_ZERO
				fmt.Println(colorRedYellow("Error: Division by zero in DIVA."))
				pc += 2
			} else {
				regs[gr] /= m
				ofr1 := 0
				ofr2 := 0
				if regs[gr] > MAX_SIGNED {
					ofr1 = FR_OVER
				}
				if regs[gr] < MIN_SIGNED {
					ofr2 = FR_OVER
				}
				regs[gr] &= 0xffff
				fr = getFlag(regs[gr]) | ofr1 | ofr2
				pc += 2
			}
		} else {
			regs[gr] = signed(regs[gr])
			regs[xr] = signed(regs[xr])
			if regs[xr] == 0 {
				fr = FR_OVER | FR_ZERO
				fmt.Println(colorRedYellow("Error: Division by zero in DIVA."))
				pc++
			} else {
				regs[gr] /= regs[xr]
				ofr1 := 0
				ofr2 := 0
				if regs[gr] > MAX_SIGNED {
					ofr1 = FR_OVER
				}
				if regs[gr] < MIN_SIGNED {
					ofr2 = FR_OVER
				}
				regs[gr] &= 0xffff
				regs[xr] &= 0xffff
				fr = getFlag(regs[gr]) | ofr1 | ofr2
				pc++
			}
		}

	case "DIVL":
		if !grIsGr.MatchString(opr) {
			m := memGet(memory, eadr)
			if m == 0 {
				fr = FR_OVER | FR_ZERO
				fmt.Println(colorRedYellow("Error: Division by zero in DIVL."))
				pc += 2
			} else {
				regs[gr] /= m
				ofr1 := 0
				ofr2 := 0
				if regs[gr] > 0xffff {
					ofr1 = FR_OVER
				}
				if regs[gr] < 0 {
					ofr2 = FR_OVER
				}
				regs[gr] &= 0xffff
				fr = getFlag(regs[gr]) | ofr1 | ofr2
				pc += 2
			}
		} else {
			if regs[xr] == 0 {
				fr = FR_OVER | FR_ZERO
				fmt.Println(colorRedYellow("Error: Division by zero in DIVL."))
				pc++
			} else {
				regs[gr] /= regs[xr]
				ofr1 := 0
				ofr2 := 0
				if regs[gr] > 0xffff {
					ofr1 = FR_OVER
				}
				if regs[gr] < 0 {
					ofr2 = FR_OVER
				}
				regs[gr] &= 0xffff
				regs[xr] &= 0xffff
				fr = getFlag(regs[gr]) | ofr1 | ofr2
				pc++
			}
		}

	case "AND":
		if !grIsGr.MatchString(opr) {
			regs[gr] &= memGet(memory, eadr)
			fr = getFlag(regs[gr])
			pc += 2
		} else {
			regs[gr] &= regs[xr]
			fr = getFlag(regs[gr])
			pc++
		}

	case "OR":
		if !grIsGr.MatchString(opr) {
			regs[gr] |= memGet(memory, eadr)
			fr = getFlag(regs[gr])
			pc += 2
		} else {
			regs[gr] |= regs[xr]
			fr = getFlag(regs[gr])
			pc++
		}

	case "XOR":
		if !grIsGr.MatchString(opr) {
			regs[gr] ^= memGet(memory, eadr)
			fr = getFlag(regs[gr])
			pc += 2
		} else {
			regs[gr] ^= regs[xr]
			fr = getFlag(regs[gr])
			pc++
		}

	case "CPA":
		if !grIsGr.MatchString(opr) {
			val = signed(regs[gr]) - signed(memGet(memory, eadr))
			if val > MAX_SIGNED {
				val = MAX_SIGNED
			}
			if val < MIN_SIGNED {
				val = MIN_SIGNED
			}
			fr = getFlag(unsigned(val))
			pc += 2
		} else {
			val = signed(regs[gr]) - signed(regs[xr])
			if val > MAX_SIGNED {
				val = MAX_SIGNED
			}
			if val < MIN_SIGNED {
				val = MIN_SIGNED
			}
			fr = getFlag(unsigned(val))
			pc++
		}

	case "CPL":
		if !grIsGr.MatchString(opr) {
			val = regs[gr] - memGet(memory, eadr)
			if val > MAX_SIGNED {
				val = MAX_SIGNED
			}
			if val < MIN_SIGNED {
				val = MIN_SIGNED
			}
			fr = getFlag(unsigned(val))
			pc += 2
		} else {
			val = regs[gr] - regs[xr]
			if val > MAX_SIGNED {
				val = MAX_SIGNED
			}
			if val < MIN_SIGNED {
				val = MIN_SIGNED
			}
			fr = getFlag(unsigned(val))
			pc++
		}

	case "SLA":
		val = regs[gr] & 0x8000
		regs[gr] <<= eadr
		ofr := regs[gr] & 0x8000
		ofr >>= 13
		regs[gr] |= val
		regs[gr] &= 0xffff
		fr = getFlag(regs[gr]) | ofr
		pc += 2

	case "SRA":
		val = regs[gr]
		ofr := regs[gr] & (0x0001 << (eadr - 1))
		ofr <<= (2 - (eadr - 1))
		if val&0x8000 != 0 {
			val &= 0x7fff
			val >>= eadr
			val += ((0x7fff >> eadr) ^ 0xffff)
		} else {
			val >>= eadr
		}
		regs[gr] = val
		fr = getFlag(regs[gr]) | ofr
		pc += 2

	case "SLL":
		regs[gr] <<= eadr
		ofr := regs[gr] & 0x10000
		ofr >>= 14
		regs[gr] &= 0xffff
		fr = getFlag(regs[gr]) | ofr
		pc += 2

	case "SRL":
		ofr := regs[gr] & (0x0001 << (eadr - 1))
		ofr <<= 2 - (eadr - 1)
		regs[gr] >>= eadr
		fr = getFlag(regs[gr]) | ofr
		pc += 2

	case "JMI":
		if (fr & FR_MINUS) == FR_MINUS {
			pc = eadr
		} else {
			pc += 2
		}

	case "JNZ":
		if (fr & FR_ZERO) != FR_ZERO {
			pc = eadr
		} else {
			pc += 2
		}

	case "JZE":
		if (fr & FR_ZERO) == FR_ZERO {
			pc = eadr
		} else {
			pc += 2
		}

	case "JUMP":
		pc = eadr

	case "JPL":
		if ((fr & FR_MINUS) != FR_MINUS) && ((fr & FR_ZERO) != FR_ZERO) {
			pc = eadr
		} else {
			pc += 2
		}

	case "JOV":
		if (fr & FR_OVER) != 0 {
			pc = eadr
		} else {
			pc += 2
		}

	case "PUSH":
		sp--
		if sp <= addressMax {
			return false, fmt.Errorf("Stack overflow at #%s: SP = #%s", hex(pc, 4), hex(sp, 4))
		}
		memPut(memory, sp, eadr)
		pc += 2

	case "POP":
		regs[gr] = memGet(memory, sp)
		sp++
		if sp > STACK_TOP {
			return false, fmt.Errorf("Stack underflow at #%s: SP = #%s", hex(pc, 4), hex(sp, 4))
		}
		pc++

	case "CALL":
		sp--
		if sp <= addressMax {
			return false, fmt.Errorf("Stack overflow at #%s: SP = #%s", hex(pc, 4), hex(sp, 4))
		}
		memPut(memory, sp, pc+2)
		pc = eadr

	case "RET":
		pc = memGet(memory, sp)
		sp++
		if sp > STACK_TOP {
			return false, fmt.Errorf("Program finished (RET)")
		}

	case "SVC":
		switch eadr {
		case SYS_IN:
			inputMode = INPUT_MODE_IN
			stopFlag = true
		case SYS_OUT:
			execOut(memory, state)
			pc += 2
		case EXIT_USR:
			return false, fmt.Errorf("Program finished (SVC %d)", EXIT_USR)
		case EXIT_OVF:
			return false, fmt.Errorf("Program finished (SVC %d)", EXIT_OVF)
		case EXIT_DVZ:
			return false, fmt.Errorf("Program finished (SVC %d)", EXIT_DVZ)
		case EXIT_ROV:
			return false, fmt.Errorf("Program finished (SVC %d)", EXIT_ROV)
		}

	case "NOP":
		pc++

	default:
		return false, fmt.Errorf("Illegal instruction %s at #%s", inst, hex(pc, 4))
	}

	// Update state
	state[PC] = pc
	state[FR] = fr
	state[SP] = sp
	for i := 0; i < 8; i++ {
		state[GR0+i] = regs[i]
	}

	return stopFlag, nil
}
