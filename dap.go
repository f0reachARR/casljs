package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const dapThreadID = 1

type dapRequest struct {
	Seq       int             `json:"seq"`
	Type      string          `json:"type"`
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments"`
}

type dapProtocol struct {
	reader *bufio.Reader
	writer io.Writer
	seq    int
	mutex  sync.Mutex
}

type dapAdapter struct {
	protocol    *dapProtocol
	source      string
	asm         *AssemblerState
	memory      []uint16
	state       []int
	breakpoints map[int]bool
	input       *bufio.Scanner
	mutex       sync.Mutex
	cancel      chan string
}

func serveDAP(port int, source string, asm *AssemblerState, memory []uint16, machineState []int) error {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("cannot start debug adapter: %w", err)
	}
	defer listener.Close()

	for {
		connection, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("cannot accept debug adapter connection: %w", err)
		}
		adapter := newDAPAdapter(connection, connection, source, asm, memory, machineState)
		finished, serveErr := adapter.serve()
		connection.Close()
		if serveErr != nil {
			return serveErr
		}
		if finished {
			return nil
		}
	}
}

func newDAPAdapter(reader io.Reader, writer io.Writer, source string, asm *AssemblerState, memory []uint16, machineState []int) *dapAdapter {
	absoluteSource, err := filepath.Abs(source)
	if err == nil {
		source = absoluteSource
	}
	return &dapAdapter{
		protocol:    &dapProtocol{reader: bufio.NewReader(reader), writer: writer},
		source:      filepath.Clean(source),
		asm:         asm,
		memory:      memory,
		state:       machineState,
		breakpoints: make(map[int]bool),
		input:       bufio.NewScanner(os.Stdin),
	}
}

func (adapter *dapAdapter) serve() (bool, error) {
	for {
		request, err := adapter.protocol.readRequest()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		if request.Type != "request" {
			continue
		}
		keepServing, err := adapter.handle(request)
		if err != nil {
			if sendErr := adapter.protocol.respond(request, false, nil, err.Error()); sendErr != nil {
				return false, sendErr
			}
		}
		if !keepServing {
			return true, nil
		}
	}
}

func (adapter *dapAdapter) handle(request dapRequest) (bool, error) {
	switch request.Command {
	case "initialize":
		body := map[string]interface{}{
			"supportsConfigurationDoneRequest": true,
			"supportsTerminateRequest":         true,
		}
		return true, adapter.protocol.respond(request, true, body, "")
	case "launch":
		if err := adapter.protocol.respond(request, true, nil, ""); err != nil {
			return true, err
		}
		return true, adapter.protocol.event("initialized", nil)
	case "setBreakpoints":
		return true, adapter.setBreakpoints(request)
	case "configurationDone":
		if err := adapter.protocol.respond(request, true, nil, ""); err != nil {
			return true, err
		}
		return true, adapter.stopped("entry")
	case "threads":
		body := map[string]interface{}{"threads": []interface{}{
			map[string]interface{}{"id": dapThreadID, "name": "COMET II"},
		}}
		return true, adapter.protocol.respond(request, true, body, "")
	case "stackTrace":
		return true, adapter.stackTrace(request)
	case "scopes":
		body := map[string]interface{}{"scopes": []interface{}{
			map[string]interface{}{
				"name":               "Registers",
				"variablesReference": 1,
				"expensive":          false,
			},
		}}
		return true, adapter.protocol.respond(request, true, body, "")
	case "variables":
		return true, adapter.variables(request)
	case "continue":
		if err := adapter.protocol.respond(request, true, map[string]bool{"allThreadsContinued": true}, ""); err != nil {
			return true, err
		}
		return true, adapter.startExecution(adapter.continueExecution)
	case "next", "stepIn":
		if err := adapter.protocol.respond(request, true, nil, ""); err != nil {
			return true, err
		}
		return true, adapter.startExecution(adapter.stepSourceLine)
	case "pause":
		if err := adapter.protocol.respond(request, true, nil, ""); err != nil {
			return true, err
		}
		adapter.cancelExecution("pause")
		return true, nil
	case "disconnect", "terminate":
		adapter.cancelExecution("disconnect")
		return false, adapter.protocol.respond(request, true, nil, "")
	default:
		return true, fmt.Errorf("unsupported request %q", request.Command)
	}
}

func (adapter *dapAdapter) setBreakpoints(request dapRequest) error {
	var arguments struct {
		Source struct {
			Path string `json:"path"`
		} `json:"source"`
		Breakpoints []struct {
			Line int `json:"line"`
		} `json:"breakpoints"`
	}
	if err := json.Unmarshal(request.Arguments, &arguments); err != nil {
		return fmt.Errorf("invalid setBreakpoints arguments: %w", err)
	}

	adapter.breakpoints = make(map[int]bool)
	results := make([]interface{}, 0, len(arguments.Breakpoints))
	sourceMatches := sameFile(arguments.Source.Path, adapter.source)
	for _, requested := range arguments.Breakpoints {
		address, line, verified := adapter.addressForLine(requested.Line)
		if !sourceMatches {
			verified = false
		}
		if verified {
			adapter.breakpoints[address] = true
		}
		result := map[string]interface{}{"verified": verified, "line": line}
		if !verified {
			result["message"] = "no executable code at this line"
		}
		results = append(results, result)
	}
	return adapter.protocol.respond(request, true, map[string]interface{}{"breakpoints": results}, "")
}

func (adapter *dapAdapter) addressForLine(requestedLine int) (int, int, bool) {
	bestAddress := -1
	bestLine := 0
	for address := 0; address < addressMax; address++ {
		entry, ok := adapter.asm.memory[address]
		if !ok || !sameFile(entry.File, adapter.source) || entry.Line < requestedLine {
			continue
		}
		if bestAddress == -1 || entry.Line < bestLine || (entry.Line == bestLine && address < bestAddress) {
			bestAddress = address
			bestLine = entry.Line
		}
	}
	return bestAddress, bestLine, bestAddress >= 0
}

func (adapter *dapAdapter) startExecution(execute func(<-chan string) error) error {
	adapter.mutex.Lock()
	if adapter.cancel != nil {
		adapter.mutex.Unlock()
		return fmt.Errorf("program is already running")
	}
	cancel := make(chan string, 1)
	adapter.cancel = cancel
	adapter.mutex.Unlock()

	go func() {
		_ = execute(cancel)
		adapter.mutex.Lock()
		if adapter.cancel == cancel {
			adapter.cancel = nil
		}
		adapter.mutex.Unlock()
	}()
	return nil
}

func (adapter *dapAdapter) cancelExecution(reason string) {
	adapter.mutex.Lock()
	cancel := adapter.cancel
	adapter.mutex.Unlock()
	if cancel != nil {
		select {
		case cancel <- reason:
		default:
		}
	}
}

func (adapter *dapAdapter) continueExecution(cancel <-chan string) error {
	for {
		if stopped, err := adapter.cancelled(cancel); stopped {
			return err
		}
		terminated, err := adapter.executeInstruction()
		if err != nil {
			return adapter.executionError(err)
		}
		if terminated {
			return adapter.protocol.event("terminated", nil)
		}
		if adapter.breakpoints[adapter.state[PC]] {
			return adapter.stopped("breakpoint")
		}
	}
}

func (adapter *dapAdapter) stepSourceLine(cancel <-chan string) error {
	startFile, startLine := adapter.sourceLocation(adapter.state[PC])
	for {
		if stopped, err := adapter.cancelled(cancel); stopped {
			return err
		}
		terminated, err := adapter.executeInstruction()
		if err != nil {
			return adapter.executionError(err)
		}
		if terminated {
			return adapter.protocol.event("terminated", nil)
		}
		file, line := adapter.sourceLocation(adapter.state[PC])
		if file != startFile || line != startLine {
			return adapter.stopped("step")
		}
	}
}

func (adapter *dapAdapter) cancelled(cancel <-chan string) (bool, error) {
	select {
	case reason := <-cancel:
		if reason == "disconnect" {
			return true, nil
		}
		return true, adapter.stopped(reason)
	default:
		return false, nil
	}
}

func (adapter *dapAdapter) executeInstruction() (bool, error) {
	stopForInput, err := stepExec(adapter.memory, adapter.state)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Program finished") {
			return true, nil
		}
		return false, err
	}
	if stopForInput {
		var text string
		if len(inputBuffer) > 0 {
			text = inputBuffer[0]
			inputBuffer = inputBuffer[1:]
		} else {
			if !adapter.input.Scan() {
				return true, nil
			}
			text = adapter.input.Text()
		}
		execIn(adapter.memory, adapter.state, text)
		inputMode = INPUT_MODE_CMD
	}
	return false, nil
}

func (adapter *dapAdapter) executionError(err error) error {
	if sendErr := adapter.protocol.event("output", map[string]interface{}{
		"category": "stderr",
		"output":   err.Error() + "\n",
	}); sendErr != nil {
		return sendErr
	}
	return adapter.stopped("exception")
}

func (adapter *dapAdapter) stopped(reason string) error {
	return adapter.protocol.event("stopped", map[string]interface{}{
		"reason":            reason,
		"threadId":          dapThreadID,
		"allThreadsStopped": true,
	})
}

func (adapter *dapAdapter) stackTrace(request dapRequest) error {
	file, line := adapter.sourceLocation(adapter.state[PC])
	frame := map[string]interface{}{
		"id":     1,
		"name":   fmt.Sprintf("COMET II at #%04x", adapter.state[PC]),
		"line":   line,
		"column": 1,
		"source": map[string]string{"name": filepath.Base(file), "path": file},
	}
	body := map[string]interface{}{"stackFrames": []interface{}{frame}, "totalFrames": 1}
	return adapter.protocol.respond(request, true, body, "")
}

func (adapter *dapAdapter) variables(request dapRequest) error {
	var arguments struct {
		VariablesReference int `json:"variablesReference"`
	}
	if err := json.Unmarshal(request.Arguments, &arguments); err != nil {
		return fmt.Errorf("invalid variables arguments: %w", err)
	}
	variables := make([]interface{}, 0, 11)
	if arguments.VariablesReference == 1 {
		names := []string{"PC", "FR", "GR0", "GR1", "GR2", "GR3", "GR4", "GR5", "GR6", "GR7", "SP"}
		for index, name := range names {
			value := adapter.state[index]
			variables = append(variables, map[string]interface{}{
				"name":               name,
				"value":              fmt.Sprintf("#%04x (%d)", value&0xffff, signed(value)),
				"variablesReference": 0,
			})
		}
	}
	return adapter.protocol.respond(request, true, map[string]interface{}{"variables": variables}, "")
}

func (adapter *dapAdapter) sourceLocation(address int) (string, int) {
	if entry, ok := adapter.asm.memory[address]; ok {
		file := entry.File
		absoluteFile, err := filepath.Abs(file)
		if err == nil {
			file = absoluteFile
		}
		return filepath.Clean(file), entry.Line
	}
	return adapter.source, 1
}

func sameFile(left, right string) bool {
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbsolute
	}
	if rightErr == nil {
		right = rightAbsolute
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func (protocol *dapProtocol) readRequest() (dapRequest, error) {
	contentLength := -1
	for {
		line, err := protocol.reader.ReadString('\n')
		if err != nil {
			return dapRequest{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if found && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil || contentLength < 0 {
				return dapRequest{}, fmt.Errorf("invalid Content-Length")
			}
		}
	}
	if contentLength < 0 {
		return dapRequest{}, fmt.Errorf("missing Content-Length")
	}
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(protocol.reader, content); err != nil {
		return dapRequest{}, err
	}
	var request dapRequest
	if err := json.Unmarshal(content, &request); err != nil {
		return dapRequest{}, fmt.Errorf("invalid DAP message: %w", err)
	}
	return request, nil
}

func (protocol *dapProtocol) respond(request dapRequest, success bool, body interface{}, message string) error {
	response := map[string]interface{}{
		"type":        "response",
		"request_seq": request.Seq,
		"success":     success,
		"command":     request.Command,
	}
	if body != nil {
		response["body"] = body
	}
	if message != "" {
		response["message"] = message
	}
	return protocol.send(response)
}

func (protocol *dapProtocol) event(event string, body interface{}) error {
	message := map[string]interface{}{"type": "event", "event": event}
	if body != nil {
		message["body"] = body
	}
	return protocol.send(message)
}

func (protocol *dapProtocol) send(message map[string]interface{}) error {
	protocol.mutex.Lock()
	defer protocol.mutex.Unlock()
	protocol.seq++
	message["seq"] = protocol.seq
	content, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(protocol.writer, "Content-Length: %d\r\n\r\n%s", len(content), content)
	return err
}
