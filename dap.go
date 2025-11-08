package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// DAP Protocol Messages
// Based on Debug Adapter Protocol specification

// ProtocolMessage is the base message type
type ProtocolMessage struct {
	Seq  int    `json:"seq"`
	Type string `json:"type"`
}

// Request message
type Request struct {
	ProtocolMessage
	Command   string                 `json:"command"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Response message
type Response struct {
	ProtocolMessage
	RequestSeq int                    `json:"request_seq"`
	Success    bool                   `json:"success"`
	Command    string                 `json:"command"`
	Message    string                 `json:"message,omitempty"`
	Body       map[string]interface{} `json:"body,omitempty"`
}

// Event message
type Event struct {
	ProtocolMessage
	Event string                 `json:"event"`
	Body  map[string]interface{} `json:"body,omitempty"`
}

// DAPServer implements Debug Adapter Protocol
type DAPServer struct {
	conn         net.Conn
	reader       *bufio.Reader
	seq          int
	mu           sync.Mutex
	memory       []uint16
	state        []int
	breakpoints  map[int]bool
	running      bool
	stopOnEntry  bool
	terminated   bool
	asmState     *AssemblerState
	sourceFile   string
}

// NewDAPServer creates a new DAP server instance
func NewDAPServer(conn net.Conn) *DAPServer {
	return &DAPServer{
		conn:        conn,
		reader:      bufio.NewReader(conn),
		seq:         1,
		breakpoints: make(map[int]bool),
	}
}

// Start begins processing DAP messages
func (d *DAPServer) Start() {
	defer d.conn.Close()

	for !d.terminated {
		msg, err := d.readMessage()
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "DAP read error: %v\n", err)
			}
			break
		}

		d.handleMessage(msg)
	}
}

// readMessage reads a single DAP message
func (d *DAPServer) readMessage() (map[string]interface{}, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := d.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Read content
	contentLength := 0
	if lenStr, ok := headers["Content-Length"]; ok {
		contentLength, _ = strconv.Atoi(lenStr)
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing or invalid Content-Length")
	}

	content := make([]byte, contentLength)
	_, err := io.ReadFull(d.reader, content)
	if err != nil {
		return nil, err
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// sendMessage sends a DAP message
func (d *DAPServer) sendMessage(msg interface{}) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	_, err = d.conn.Write([]byte(header))
	if err != nil {
		return err
	}

	_, err = d.conn.Write(content)
	return err
}

// sendResponse sends a response message
func (d *DAPServer) sendResponse(requestSeq int, command string, success bool, message string, body map[string]interface{}) {
	d.mu.Lock()
	seq := d.seq
	d.seq++
	d.mu.Unlock()

	resp := Response{
		ProtocolMessage: ProtocolMessage{
			Seq:  seq,
			Type: "response",
		},
		RequestSeq: requestSeq,
		Success:    success,
		Command:    command,
		Message:    message,
		Body:       body,
	}

	d.sendMessage(resp)
}

// sendEvent sends an event message
func (d *DAPServer) sendEvent(event string, body map[string]interface{}) {
	d.mu.Lock()
	seq := d.seq
	d.seq++
	d.mu.Unlock()

	evt := Event{
		ProtocolMessage: ProtocolMessage{
			Seq:  seq,
			Type: "event",
		},
		Event: event,
		Body:  body,
	}

	d.sendMessage(evt)
}

// handleMessage processes a DAP message
func (d *DAPServer) handleMessage(msg map[string]interface{}) {
	msgType, _ := msg["type"].(string)
	if msgType != "request" {
		return
	}

	seq := int(msg["seq"].(float64))
	command, _ := msg["command"].(string)
	args, _ := msg["arguments"].(map[string]interface{})

	switch command {
	case "initialize":
		d.handleInitialize(seq, args)
	case "launch":
		d.handleLaunch(seq, args)
	case "attach":
		d.handleAttach(seq, args)
	case "setBreakpoints":
		d.handleSetBreakpoints(seq, args)
	case "configurationDone":
		d.handleConfigurationDone(seq, args)
	case "threads":
		d.handleThreads(seq, args)
	case "stackTrace":
		d.handleStackTrace(seq, args)
	case "scopes":
		d.handleScopes(seq, args)
	case "variables":
		d.handleVariables(seq, args)
	case "continue":
		d.handleContinue(seq, args)
	case "next":
		d.handleNext(seq, args)
	case "stepIn":
		d.handleStepIn(seq, args)
	case "stepOut":
		d.handleStepOut(seq, args)
	case "pause":
		d.handlePause(seq, args)
	case "disconnect":
		d.handleDisconnect(seq, args)
	case "terminate":
		d.handleTerminate(seq, args)
	default:
		d.sendResponse(seq, command, false, fmt.Sprintf("Unknown command: %s", command), nil)
	}
}

// handleInitialize handles the initialize request
func (d *DAPServer) handleInitialize(seq int, args map[string]interface{}) {
	body := map[string]interface{}{
		"supportsConfigurationDoneRequest": true,
		"supportsTerminateRequest":         true,
		"supportsRestartRequest":           false,
		"supportsCancelRequest":            false,
	}
	d.sendResponse(seq, "initialize", true, "", body)
	d.sendEvent("initialized", nil)
}

// handleLaunch handles the launch request
func (d *DAPServer) handleLaunch(seq int, args map[string]interface{}) {
	program, ok := args["program"].(string)
	if !ok {
		d.sendResponse(seq, "launch", false, "Missing 'program' argument", nil)
		return
	}

	d.sourceFile = program

	// Check for stopOnEntry
	if stopOnEntry, ok := args["stopOnEntry"].(bool); ok {
		d.stopOnEntry = stopOnEntry
	}

	// Assemble the program
	asmState := newAssemblerState()
	comet2bin, startLabel, err := assemble(program, asmState)
	if err != nil {
		d.sendResponse(seq, "launch", false, fmt.Sprintf("Assembly failed: %v", err), nil)
		return
	}

	d.asmState = asmState

	// Initialize COMET2
	d.memory = make([]uint16, 0x10000)
	copy(d.memory, comet2bin)
	startAddress := uint16(expandLabel(asmState.symtbl, startLabel))

	d.state = []int{int(startAddress), FR_PLUS, 0, 0, 0, 0, 0, 0, 0, 0, STACK_TOP}

	d.sendResponse(seq, "launch", true, "", nil)
}

// handleAttach handles the attach request
func (d *DAPServer) handleAttach(seq int, args map[string]interface{}) {
	d.sendResponse(seq, "attach", false, "Attach not supported", nil)
}

// handleSetBreakpoints handles the setBreakpoints request
func (d *DAPServer) handleSetBreakpoints(seq int, args map[string]interface{}) {
	_, _ = args["source"].(map[string]interface{})
	breakpointsArg, _ := args["breakpoints"].([]interface{})

	// Clear existing breakpoints
	d.breakpoints = make(map[int]bool)

	// Set new breakpoints
	verifiedBreakpoints := []map[string]interface{}{}
	for _, bp := range breakpointsArg {
		bpMap, _ := bp.(map[string]interface{})
		line := int(bpMap["line"].(float64))

		// Find the address for this line
		address := d.findAddressForLine(line)
		if address >= 0 {
			d.breakpoints[address] = true
			verifiedBreakpoints = append(verifiedBreakpoints, map[string]interface{}{
				"verified": true,
				"line":     line,
			})
		} else {
			verifiedBreakpoints = append(verifiedBreakpoints, map[string]interface{}{
				"verified": false,
				"line":     line,
			})
		}
	}

	body := map[string]interface{}{
		"breakpoints": verifiedBreakpoints,
	}
	d.sendResponse(seq, "setBreakpoints", true, "", body)
}

// findAddressForLine finds the memory address for a source line
func (d *DAPServer) findAddressForLine(line int) int {
	if d.asmState == nil {
		return -1
	}

	// Search through memory entries to find the address for this line
	for addr, entry := range d.asmState.memory {
		if entry.Line == line {
			return addr
		}
	}

	return -1
}

// handleConfigurationDone handles the configurationDone request
func (d *DAPServer) handleConfigurationDone(seq int, args map[string]interface{}) {
	d.sendResponse(seq, "configurationDone", true, "", nil)

	if d.stopOnEntry {
		// Send stopped event
		d.sendEvent("stopped", map[string]interface{}{
			"reason":      "entry",
			"threadId":    1,
			"allThreadsStopped": true,
		})
	} else {
		// Continue execution
		go d.runProgram()
	}
}

// handleThreads handles the threads request
func (d *DAPServer) handleThreads(seq int, args map[string]interface{}) {
	threads := []map[string]interface{}{
		{
			"id":   1,
			"name": "COMET2",
		},
	}
	body := map[string]interface{}{
		"threads": threads,
	}
	d.sendResponse(seq, "threads", true, "", body)
}

// handleStackTrace handles the stackTrace request
func (d *DAPServer) handleStackTrace(seq int, args map[string]interface{}) {
	frames := []map[string]interface{}{
		{
			"id":     1,
			"name":   "main",
			"line":   d.findLineForAddress(d.state[PC]),
			"column": 0,
			"source": map[string]interface{}{
				"name": d.sourceFile,
				"path": d.sourceFile,
			},
		},
	}

	body := map[string]interface{}{
		"stackFrames": frames,
		"totalFrames": 1,
	}
	d.sendResponse(seq, "stackTrace", true, "", body)
}

// findLineForAddress finds the source line for a memory address
func (d *DAPServer) findLineForAddress(address int) int {
	if d.asmState == nil {
		return 0
	}

	if entry, ok := d.asmState.memory[address]; ok {
		return entry.Line
	}

	return 0
}

// handleScopes handles the scopes request
func (d *DAPServer) handleScopes(seq int, args map[string]interface{}) {
	scopes := []map[string]interface{}{
		{
			"name":               "Registers",
			"variablesReference": 1,
			"expensive":          false,
		},
	}

	body := map[string]interface{}{
		"scopes": scopes,
	}
	d.sendResponse(seq, "scopes", true, "", body)
}

// handleVariables handles the variables request
func (d *DAPServer) handleVariables(seq int, args map[string]interface{}) {
	variables := []map[string]interface{}{
		{"name": "PC", "value": fmt.Sprintf("#%04X (%d)", d.state[PC], d.state[PC]), "variablesReference": 0},
		{"name": "FR", "value": fmt.Sprintf("%d", d.state[FR]), "variablesReference": 0},
		{"name": "GR0", "value": fmt.Sprintf("#%04X (%d)", d.state[GR0], signed(d.state[GR0])), "variablesReference": 0},
		{"name": "GR1", "value": fmt.Sprintf("#%04X (%d)", d.state[GR1], signed(d.state[GR1])), "variablesReference": 0},
		{"name": "GR2", "value": fmt.Sprintf("#%04X (%d)", d.state[GR2], signed(d.state[GR2])), "variablesReference": 0},
		{"name": "GR3", "value": fmt.Sprintf("#%04X (%d)", d.state[GR3], signed(d.state[GR3])), "variablesReference": 0},
		{"name": "GR4", "value": fmt.Sprintf("#%04X (%d)", d.state[GR4], signed(d.state[GR4])), "variablesReference": 0},
		{"name": "GR5", "value": fmt.Sprintf("#%04X (%d)", d.state[GR5], signed(d.state[GR5])), "variablesReference": 0},
		{"name": "GR6", "value": fmt.Sprintf("#%04X (%d)", d.state[GR6], signed(d.state[GR6])), "variablesReference": 0},
		{"name": "GR7", "value": fmt.Sprintf("#%04X (%d)", d.state[GR7], signed(d.state[GR7])), "variablesReference": 0},
		{"name": "SP", "value": fmt.Sprintf("#%04X (%d)", d.state[SP], d.state[SP]), "variablesReference": 0},
	}

	body := map[string]interface{}{
		"variables": variables,
	}
	d.sendResponse(seq, "variables", true, "", body)
}

// handleContinue handles the continue request
func (d *DAPServer) handleContinue(seq int, args map[string]interface{}) {
	d.sendResponse(seq, "continue", true, "", map[string]interface{}{
		"allThreadsContinued": true,
	})

	go d.runProgram()
}

// handleNext handles the next (step over) request
func (d *DAPServer) handleNext(seq int, args map[string]interface{}) {
	d.sendResponse(seq, "next", true, "", nil)

	go func() {
		stopFlag, err := stepExec(d.memory, d.state)
		if err != nil {
			d.sendEvent("stopped", map[string]interface{}{
				"reason":            "exception",
				"description":       err.Error(),
				"threadId":          1,
				"allThreadsStopped": true,
			})
			return
		}

		if stopFlag {
			// Waiting for input
			d.sendEvent("stopped", map[string]interface{}{
				"reason":            "pause",
				"description":       "Waiting for input",
				"threadId":          1,
				"allThreadsStopped": true,
			})
		} else {
			d.sendEvent("stopped", map[string]interface{}{
				"reason":            "step",
				"threadId":          1,
				"allThreadsStopped": true,
			})
		}
	}()
}

// handleStepIn handles the stepIn request
func (d *DAPServer) handleStepIn(seq int, args map[string]interface{}) {
	// For COMET2, stepIn is the same as next
	d.handleNext(seq, args)
}

// handleStepOut handles the stepOut request
func (d *DAPServer) handleStepOut(seq int, args map[string]interface{}) {
	d.sendResponse(seq, "stepOut", true, "", nil)

	go func() {
		// Step out means continue until RET
		for {
			inst, _, _ := parse(d.memory, d.state)
			stopFlag, err := stepExec(d.memory, d.state)
			
			if err != nil {
				d.sendEvent("stopped", map[string]interface{}{
					"reason":            "exception",
					"description":       err.Error(),
					"threadId":          1,
					"allThreadsStopped": true,
				})
				return
			}

			if stopFlag {
				d.sendEvent("stopped", map[string]interface{}{
					"reason":            "pause",
					"description":       "Waiting for input",
					"threadId":          1,
					"allThreadsStopped": true,
				})
				return
			}

			if inst == "RET" {
				break
			}

			// Check breakpoints
			if d.breakpoints[d.state[PC]] {
				break
			}
		}

		d.sendEvent("stopped", map[string]interface{}{
			"reason":            "step",
			"threadId":          1,
			"allThreadsStopped": true,
		})
	}()
}

// handlePause handles the pause request
func (d *DAPServer) handlePause(seq int, args map[string]interface{}) {
	d.running = false
	d.sendResponse(seq, "pause", true, "", nil)
	d.sendEvent("stopped", map[string]interface{}{
		"reason":            "pause",
		"threadId":          1,
		"allThreadsStopped": true,
	})
}

// handleDisconnect handles the disconnect request
func (d *DAPServer) handleDisconnect(seq int, args map[string]interface{}) {
	d.terminated = true
	d.sendResponse(seq, "disconnect", true, "", nil)
	d.sendEvent("terminated", nil)
}

// handleTerminate handles the terminate request
func (d *DAPServer) handleTerminate(seq int, args map[string]interface{}) {
	d.terminated = true
	d.sendResponse(seq, "terminate", true, "", nil)
	d.sendEvent("terminated", nil)
}

// runProgram continues execution until a breakpoint or error
func (d *DAPServer) runProgram() {
	d.running = true

	for d.running {
		// Check breakpoint before execution
		if d.breakpoints[d.state[PC]] {
			d.running = false
			d.sendEvent("stopped", map[string]interface{}{
				"reason":            "breakpoint",
				"threadId":          1,
				"allThreadsStopped": true,
			})
			return
		}

		stopFlag, err := stepExec(d.memory, d.state)
		if err != nil {
			d.running = false
			if strings.Contains(err.Error(), "Program finished") {
				d.sendEvent("terminated", nil)
			} else {
				d.sendEvent("stopped", map[string]interface{}{
					"reason":            "exception",
					"description":       err.Error(),
					"threadId":          1,
					"allThreadsStopped": true,
				})
			}
			return
		}

		if stopFlag {
			// Waiting for input
			d.running = false
			d.sendEvent("stopped", map[string]interface{}{
				"reason":            "pause",
				"description":       "Waiting for input (use stdin)",
				"threadId":          1,
				"allThreadsStopped": true,
			})
			return
		}
	}
}

// StartDAPServer starts the DAP server on the specified port
func StartDAPServer(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("failed to start DAP server: %v", err)
	}
	defer listener.Close()

	fmt.Fprintf(os.Stderr, "DAP server listening on port %d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "DAP accept error: %v\n", err)
			continue
		}

		server := NewDAPServer(conn)
		go server.Start()
	}
}
