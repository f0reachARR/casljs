package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const VERSION = "1.0.4 KIT (Jan 23, 2025) - Go Edition"

// System call addresses
const (
	SYS_IN   = 0xfff0
	SYS_OUT  = 0xfff2
	EXIT_USR = 0x0000
	EXIT_OVF = 0x0001
	EXIT_DVZ = 0x0002
	EXIT_ROV = 0x0003
)

// Flag register bits
const (
	FR_PLUS  = 0
	FR_ZERO  = 1
	FR_MINUS = 2
	FR_OVER  = 4
)

// Stack configuration
const STACK_TOP = 0xff00

// Register indices
const (
	PC = iota
	FR
	GR0
	GR1
	GR2
	GR3
	GR4
	GR5
	GR6
	GR7
	SP
)

// Value limits
const (
	MAX_SIGNED = 32767
	MIN_SIGNED = -32768
)

// Input modes
const (
	INPUT_MODE_CMD = iota
	INPUT_MODE_IN
)

// Command line options
var (
	optAll      = flag.Bool("a", false, "[casl2] show detailed info")
	optCasl     = flag.Bool("c", false, "[casl2] apply casl2 only")
	optRun      = flag.Bool("r", false, "[comet2] run immediately")
	optNoColor  = flag.Bool("n", false, "[casl2/comet2] disable color messages")
	optQuiet    = flag.Bool("q", false, "[casl2/comet2] be quiet")
	optQuietRun = flag.Bool("Q", false, "[comet2] be QUIET! (implies -q and -r)")
	optVersion  = flag.Bool("V", false, "output the version number")
)

// Global variables
var (
	comet2mem          []uint16
	comet2startAddress uint16
	state              []int
	inputMode          int
	inputBuffer        []string
	lastCmd            string
	nextCmd            string
	addressMax         int
)

// Instruction table for CASL2
type InstructionType string

const (
	OP1   InstructionType = "op1"
	OP2   InstructionType = "op2"
	OP3   InstructionType = "op3"
	OP4   InstructionType = "op4"
	OP5   InstructionType = "op5"
	START InstructionType = "start"
	END   InstructionType = "end"
	DS    InstructionType = "ds"
	DC    InstructionType = "dc"
	IN    InstructionType = "in"
	OUT   InstructionType = "out"
	RPUSH InstructionType = "rpush"
	RPOP  InstructionType = "rpop"
)

type Instruction struct {
	Code uint8
	Type InstructionType
}

var CASL2TBL = map[string]Instruction{
	"NOP":   {0x00, OP4},
	"LD":    {0x10, OP5},
	"ST":    {0x11, OP1},
	"LAD":   {0x12, OP1},
	"ADDA":  {0x20, OP5},
	"SUBA":  {0x21, OP5},
	"ADDL":  {0x22, OP5},
	"SUBL":  {0x23, OP5},
	"MULA":  {0x28, OP5},
	"DIVA":  {0x29, OP5},
	"MULL":  {0x2A, OP5},
	"DIVL":  {0x2B, OP5},
	"AND":   {0x30, OP5},
	"OR":    {0x31, OP5},
	"XOR":   {0x32, OP5},
	"CPA":   {0x40, OP5},
	"CPL":   {0x41, OP5},
	"SLA":   {0x50, OP1},
	"SRA":   {0x51, OP1},
	"SLL":   {0x52, OP1},
	"SRL":   {0x53, OP1},
	"JMI":   {0x61, OP2},
	"JNZ":   {0x62, OP2},
	"JZE":   {0x63, OP2},
	"JUMP":  {0x64, OP2},
	"JPL":   {0x65, OP2},
	"JOV":   {0x66, OP2},
	"PUSH":  {0x70, OP2},
	"POP":   {0x71, OP3},
	"CALL":  {0x80, OP2},
	"RET":   {0x81, OP4},
	"SVC":   {0xf0, OP2},
	"START": {0x00, START},
	"END":   {0x00, END},
	"DS":    {0x00, DS},
	"DC":    {0x00, DC},
	"IN":    {0x00, IN},
	"OUT":   {0x00, OUT},
	"RPUSH": {0x00, RPUSH},
	"RPOP":  {0x00, RPOP},
}

// Symbol table entry
type SymbolEntry struct {
	Val  interface{}
	File string
	Line int
}

type MemoryEntry struct {
	Val  interface{}
	File string
	Line int
}

// Assembler state
type AssemblerState struct {
	symtbl         map[string]*SymbolEntry
	memory         map[int]*MemoryEntry
	buf            []string
	outdump        []string
	actualLabel    string
	virtualLabel   string
	firstStart     bool
	varScope       string
	literalCounter int
	file           string
	line           int
}

func newAssemblerState() *AssemblerState {
	return &AssemblerState{
		symtbl:     make(map[string]*SymbolEntry),
		memory:     make(map[int]*MemoryEntry),
		buf:        make([]string, 0),
		outdump:    make([]string, 0),
		firstStart: true,
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: c2c2 [options] <casl2file> [input1 ...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *optVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if *optQuietRun {
		*optQuiet = true
		*optRun = true
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "[CASL2 ERROR] No casl2 source file is specified.")
		os.Exit(1)
	}

	inputFilepath := args[0]
	inputBuffer = args[1:]

	if !*optQuiet {
		printGreen(`   _________   _____ __       ________
  / ____/   | / ___// /      /  _/  _/
 / /   / /| | \__ \/ /       / / / /  
/ /___/ ___ |___/ / /___   _/ /_/ /   
\____/_/  |_/____/_____/  /___/___/   `)
		fmt.Printf("This is CASL II, version %s.\n(c) 2001-2023, Osamu Mizuno.\n\n", VERSION)
	}

	// Assemble the code
	asmState := newAssemblerState()
	comet2bin, startLabel, err := assemble(inputFilepath, asmState)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	caslPrint("Successfully assembled.")

	if *optCasl {
		os.Exit(0)
	}

	// Initialize COMET2
	comet2mem = make([]uint16, 0x10000) // Full 64K memory space
	copy(comet2mem, comet2bin)
	comet2startAddress = uint16(expandLabel(asmState.symtbl, startLabel))

	state = []int{int(comet2startAddress), FR_PLUS, 0, 0, 0, 0, 0, 0, 0, 0, STACK_TOP}

	if !*optQuiet {
		printGreen(`   __________  __  _______________   ________
  / ____/ __ \/  |/  / ____/_  __/  /  _/  _/
 / /   / / / / /|_/ / __/   / /     / / / /  
/ /___/ /_/ / /  / / /___  / /    _/ /_/ /   
\____/\____/_/  /_/_____/ /_/    /___/___/  `)
		fmt.Printf("This is COMET II, version %s.\n(c) 2001-2023, Osamu Mizuno.\n\n", VERSION)
		cmdPrint(comet2mem, state, []string{})
	}

	if *optRun {
		nextCmd = "run"
	}

	// Main loop
	inputMode = INPUT_MODE_CMD
	scanner := bufio.NewScanner(os.Stdin)

	for {
		var cmd string

		if inputMode == INPUT_MODE_CMD {
			if nextCmd != "" {
				cmd = nextCmd
				nextCmd = ""
			} else {
				fmt.Print(colorYellow("comet2") + "> ")
				if !scanner.Scan() {
					break
				}
				cmd = strings.TrimSpace(scanner.Text())
			}

			if cmd == "" {
				cmd = lastCmd
			} else {
				lastCmd = cmd
			}

			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				continue
			}

			cmd2 := parts[0]
			args := parts[1:]

			if cmd2 == "quit" || cmd2 == "q" {
				cometPrint("[Comet2 finished]")
				break
			}

			err := executeCommand(cmd2, args, comet2mem, state)
			if err != nil {
				if strings.Contains(err.Error(), "Program finished") ||
					strings.Contains(err.Error(), "Stack overflow") ||
					strings.Contains(err.Error(), "Stack underflow") {
					fmt.Println(colorWhiteGreen(err.Error()))
					break
				}
				fmt.Fprintln(os.Stderr, colorRedYellow(err.Error()))
			}

		} else if inputMode == INPUT_MODE_IN {
			var input string
			prompt := ""
			if !*optQuietRun {
				prompt = colorIGreen("IN") + "> "
			}

			if len(inputBuffer) > 0 {
				input = inputBuffer[0]
				inputBuffer = inputBuffer[1:]
				if !*optQuietRun {
					fmt.Printf("%s%s\n", prompt, input)
				}
			} else {
				if prompt != "" {
					fmt.Print(prompt)
				}
				if !scanner.Scan() {
					break
				}
				input = scanner.Text()
			}

			execIn(comet2mem, state, input)
			inputMode = INPUT_MODE_CMD

			if !*optQuiet {
				if lastCmd == "s" || lastCmd == "step" {
					cmdPrint(comet2mem, state, []string{})
				}
			}
		}
	}
}

// Color functions
func strColor(code, str string) string {
	if *optNoColor {
		return str
	}
	return code + str + "\x1b[0m"
}

func colorGreen(str string) string {
	return strColor("\x1b[32m", str)
}

func colorIGreen(str string) string {
	return strColor("\x1b[3;32m", str)
}

func colorWhiteGreen(str string) string {
	return strColor("\x1b[37;48;5;22m", str)
}

func colorRedYellow(str string) string {
	return strColor("\x1b[31;43m", str)
}

func colorRed(str string) string {
	return strColor("\x1b[31m", str)
}

func colorIRed(str string) string {
	return strColor("\x1b[3;31m", str)
}

func colorYellow(str string) string {
	return strColor("\x1b[38;5;214m", str)
}

func colorBCyan(str string) string {
	return strColor("\x1b[1;36m", str)
}

func printGreen(str string) {
	fmt.Println(colorGreen(str))
}

func caslPrint(msg string) {
	if !*optQuiet {
		fmt.Println(msg)
	}
}

func cometPrint(msg string) {
	fmt.Println(msg)
}

func cometOut(msg string) {
	prefix := ""
	if !*optQuietRun {
		prefix = colorIRed("OUT") + "> "
	}
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Print(prefix + msg)
}

// Utility functions
func hex(val int, length int) string {
	format := fmt.Sprintf("%%0%dx", length)
	return fmt.Sprintf(format, val)
}

func spacePadding(val, length int) string {
	str := strconv.Itoa(val)
	for len(str) < length {
		str = " " + str
	}
	return str
}

func signed(val int) int {
	if val >= 32768 && val < 65536 {
		val -= 65536
	}
	return val
}

func unsigned(val int) int {
	if val >= -32768 && val < 0 {
		val += 65536
	}
	return val
}

func checkNumber(val string) bool {
	if val == "" {
		return false
	}
	if strings.HasPrefix(val, "#") {
		_, err := strconv.ParseInt(val[1:], 16, 64)
		return err == nil
	}
	_, err := strconv.ParseInt(val, 10, 64)
	return err == nil
}

func expandNumber(val string) (int, bool) {
	if !checkNumber(val) {
		return 0, false
	}
	if strings.HasPrefix(val, "#") {
		num, _ := strconv.ParseInt(val[1:], 16, 64)
		return int(num) & 0xffff, true
	}
	num, _ := strconv.ParseInt(val, 10, 64)
	return int(num) & 0xffff, true
}

func getFlag(val int) int {
	if val&0x8000 != 0 {
		return FR_MINUS
	} else if val == 0 {
		return FR_ZERO
	} else {
		return FR_PLUS
	}
}

func memGet(memory []uint16, pc int) int {
	if pc < 0 || pc >= len(memory) {
		return 0
	}
	return int(memory[pc])
}

func memPut(memory []uint16, pc int, val int) {
	if pc < 0 {
		return
	}
	
	// Ensure memory is large enough
	for len(memory) <= pc {
		// This won't work - we need to use pointers or return the slice
		// For now, just bounds check
		if pc >= len(memory) {
			return
		}
	}
	
	memory[pc] = uint16(val & 0xffff)
}
