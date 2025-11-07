package main

import (
	"fmt"
	"strconv"
)

func executeCommand(cmd string, args []string, memory []uint16, state []int) error {
	commands := map[string]func([]uint16, []int, []string) error{
		"r":    cmdRun,
		"run":  cmdRun,
		"s":    cmdStep,
		"step": cmdStep,
		"p":    cmdPrint,
		"print": cmdPrint,
		"h":    cmdHelp,
		"help": cmdHelp,
		"du":   cmdDump,
		"dump": cmdDump,
		"st":   cmdStack,
		"stack": cmdStack,
		"di":    cmdDisasm,
		"disasm": cmdDisasm,
	}

	if handler, ok := commands[cmd]; ok {
		return handler(memory, state, args)
	}

	return fmt.Errorf("Undefined command \"%s\". Try \"help\".", cmd)
}

func cmdRun(memory []uint16, state []int, args []string) error {
	nextCmd = "run"
	stopFlag, err := stepExec(memory, state)
	if err != nil {
		nextCmd = ""
		return err
	}

	if stopFlag {
		// exec_in will handle this
		return nil
	}

	// Check for breakpoints (not implemented in minimal version)
	return nil
}

func cmdStep(memory []uint16, state []int, args []string) error {
	count := 1
	if len(args) > 0 {
		if n, ok := expandNumber(args[0]); ok {
			count = n
		}
	}

	count--
	if count > 0 {
		nextCmd = fmt.Sprintf("step %d", count)
	} else {
		nextCmd = ""
	}

	_, err := stepExec(memory, state)
	if err != nil {
		return err
	}

	if !*optQuiet {
		cmdPrint(memory, state, []string{})
	}

	return nil
}

func cmdPrint(memory []uint16, state []int, args []string) error {
	pc := state[PC]
	fr := state[FR]
	sp := state[SP]
	regs := state[GR0 : GR7+1]

	// Get current instruction
	inst, opr, _ := parse(memory, state)

	cometPrint("")
	cometPrint(fmt.Sprintf("%s  %s [ %s ]",
		colorBCyan("PR"),
		colorRed("#"+hex(pc, 4)),
		colorGreen(fmt.Sprintf("%s\t\t%s", inst, opr))))

	frBin := fmt.Sprintf("%d%d%d", (fr>>2)%2, (fr>>1)%2, fr%2)
	frStr := ""
	if (fr>>2)%2 == 1 {
		frStr += "O"
	} else {
		frStr += "-"
	}
	if (fr>>1)%2 == 1 {
		frStr += "S"
	} else {
		frStr += "-"
	}
	if fr%2 == 1 {
		frStr += "Z"
	} else {
		frStr += "-"
	}

	cometPrint(fmt.Sprintf("%s  %s(%s)  %s    %s(%s)[ %s ]",
		colorBCyan("SP"),
		colorRed("#"+hex(sp, 4)),
		spacePadding(signed(sp), 6),
		colorBCyan("FR"),
		colorYellow(frBin),
		spacePadding(fr, 6),
		colorGreen(frStr)))

	cometPrint(fmt.Sprintf("%s %s(%s)  %s %s(%s)  %s %s(%s)  %s %s(%s)",
		colorBCyan("GR0"), colorRed("#"+hex(regs[0], 4)), spacePadding(signed(regs[0]), 6),
		colorBCyan("GR1"), colorRed("#"+hex(regs[1], 4)), spacePadding(signed(regs[1]), 6),
		colorBCyan("GR2"), colorRed("#"+hex(regs[2], 4)), spacePadding(signed(regs[2]), 6),
		colorBCyan("GR3"), colorRed("#"+hex(regs[3], 4)), spacePadding(signed(regs[3]), 6)))

	cometPrint(fmt.Sprintf("%s %s(%s)  %s %s(%s)  %s %s(%s)  %s %s(%s)",
		colorBCyan("GR4"), colorRed("#"+hex(regs[4], 4)), spacePadding(signed(regs[4]), 6),
		colorBCyan("GR5"), colorRed("#"+hex(regs[5], 4)), spacePadding(signed(regs[5]), 6),
		colorBCyan("GR6"), colorRed("#"+hex(regs[6], 4)), spacePadding(signed(regs[6]), 6),
		colorBCyan("GR7"), colorRed("#"+hex(regs[7], 4)), spacePadding(signed(regs[7]), 6)))

	return nil
}

func cmdDump(memory []uint16, state []int, args []string) error {
	val := state[PC]
	if len(args) > 0 {
		if n, ok := expandNumber(args[0]); ok {
			val = n
		}
	}

	for row := 0; row < 16; row++ {
		base := val + (row << 3)
		line := hex(base, 4) + ":"

		for col := 0; col < 8; col++ {
			line += " " + hex(memGet(memory, base+col), 4)
		}

		line += " "
		for col := 0; col < 8; col++ {
			c := memGet(memory, base+col) & 0xff
			if c >= 0x20 && c <= 0x7f {
				line += string(rune(c))
			} else {
				line += "."
			}
		}

		cometPrint(line)
	}

	return nil
}

func cmdStack(memory []uint16, state []int, args []string) error {
	return cmdDump(memory, state, []string{strconv.Itoa(state[SP])})
}

func cmdDisasm(memory []uint16, state []int, args []string) error {
	val := state[PC]
	if len(args) > 0 {
		if n, ok := expandNumber(args[0]); ok {
			val = n
		}
	}

	// Save original PC
	origPC := state[PC]
	state[PC] = val

	for i := 0; i < 16; i++ {
		inst, opr, size := parse(memory, state)
		cometPrint(fmt.Sprintf("#%s\t%s\t%s", hex(state[PC], 4), inst, opr))
		state[PC] += size
	}

	// Restore PC
	state[PC] = origPC

	return nil
}

func cmdHelp(memory []uint16, state []int, args []string) error {
	cometPrint("List of commands:")
	cometPrint("r,  run             \t\tStart execution of program.")
	cometPrint("s,  step  [N]       \t\tStep execution. Argument N means do this N times.")
	cometPrint("p,  print           \t\tPrint status of PC/FR/SP/GR0..GR7 registers.")
	cometPrint("du, dump [ADDRESS]  \t\tDump 128 words of memory image from specified ADDRESS.")
	cometPrint("st, stack           \t\tDump 128 words of stack image.")
	cometPrint("di, disasm [ADDRESS]\t\tDisassemble 32 words from specified ADDRESS.")
	cometPrint("h,  help            \t\tPrint list of commands.")
	cometPrint("q,  quit            \t\tExit comet2.")

	return nil
}
