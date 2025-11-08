package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestDAPProtocolBasics tests basic DAP protocol message handling
func TestDAPProtocolBasics(t *testing.T) {
	// Start a DAP server in the background
	go func() {
		StartDAPServer(4711)
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the server
	conn, err := net.Dial("tcp", "127.0.0.1:4711")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Send initialize request
	initReq := map[string]interface{}{
		"seq":     1,
		"type":    "request",
		"command": "initialize",
		"arguments": map[string]interface{}{
			"clientID":   "test",
			"adapterID":  "casl2",
			"linesStartAt1": true,
		},
	}

	if err := sendDAPMessage(conn, initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	resp, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	if resp["type"] != "response" {
		t.Errorf("Expected response type, got %v", resp["type"])
	}

	if resp["command"] != "initialize" {
		t.Errorf("Expected initialize command, got %v", resp["command"])
	}

	if !resp["success"].(bool) {
		t.Errorf("Initialize request failed")
	}

	// Read initialized event
	evt, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read initialized event: %v", err)
	}

	if evt["type"] != "event" {
		t.Errorf("Expected event type, got %v", evt["type"])
	}

	if evt["event"] != "initialized" {
		t.Errorf("Expected initialized event, got %v", evt["event"])
	}
}

// TestDAPLaunch tests launching a program via DAP
func TestDAPLaunch(t *testing.T) {
	// Create a simple test program
	testProgram := `
MAIN    START
        LD      GR0, =1
        LD      GR1, =2
        ADDA    GR0, GR1
        RET
        END
`
	testFile := "/tmp/dap_test_launch.cas"
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Start a DAP server on a different port
	go func() {
		StartDAPServer(4712)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:4712")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Initialize
	initReq := map[string]interface{}{
		"seq":     1,
		"type":    "request",
		"command": "initialize",
		"arguments": map[string]interface{}{
			"clientID":   "test",
			"adapterID":  "casl2",
		},
	}
	sendDAPMessage(conn, initReq)
	readDAPMessage(reader) // Response
	readDAPMessage(reader) // Initialized event

	// Launch
	launchReq := map[string]interface{}{
		"seq":     2,
		"type":    "request",
		"command": "launch",
		"arguments": map[string]interface{}{
			"program":     testFile,
			"stopOnEntry": true,
		},
	}
	sendDAPMessage(conn, launchReq)

	resp, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read launch response: %v", err)
	}

	if !resp["success"].(bool) {
		t.Errorf("Launch failed: %v", resp["message"])
	}
}

// TestDAPStepExecution tests step execution via DAP
func TestDAPStepExecution(t *testing.T) {
	testProgram := `
MAIN    START
        LD      GR0, =5
        LD      GR1, =10
        ADDA    GR0, GR1
        RET
        END
`
	testFile := "/tmp/dap_test_step.cas"
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	go func() {
		StartDAPServer(4713)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:4713")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Initialize
	initReq := map[string]interface{}{
		"seq":     1,
		"type":    "request",
		"command": "initialize",
		"arguments": map[string]interface{}{},
	}
	sendDAPMessage(conn, initReq)
	readDAPMessage(reader)
	readDAPMessage(reader)

	// Launch with stopOnEntry
	launchReq := map[string]interface{}{
		"seq":     2,
		"type":    "request",
		"command": "launch",
		"arguments": map[string]interface{}{
			"program":     testFile,
			"stopOnEntry": true,
		},
	}
	sendDAPMessage(conn, launchReq)
	readDAPMessage(reader) // Launch response

	// Configuration done
	configReq := map[string]interface{}{
		"seq":       3,
		"type":      "request",
		"command":   "configurationDone",
		"arguments": map[string]interface{}{},
	}
	sendDAPMessage(conn, configReq)
	readDAPMessage(reader) // Config response
	
	// Should receive a stopped event
	evt, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read stopped event: %v", err)
	}

	if evt["event"] != "stopped" {
		t.Errorf("Expected stopped event, got %v", evt["event"])
	}

	// Send next (step) request
	stepReq := map[string]interface{}{
		"seq":       4,
		"type":      "request",
		"command":   "next",
		"arguments": map[string]interface{}{
			"threadId": 1,
		},
	}
	sendDAPMessage(conn, stepReq)
	readDAPMessage(reader) // Step response

	// Should receive another stopped event
	evt, err = readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read stopped event after step: %v", err)
	}

	if evt["event"] != "stopped" {
		t.Errorf("Expected stopped event after step, got %v", evt["event"])
	}
}

// TestDAPBreakpoints tests breakpoint functionality
func TestDAPBreakpoints(t *testing.T) {
	testProgram := `
MAIN    START
        LD      GR0, =1
        LD      GR1, =2
        ADDA    GR0, GR1
        LD      GR2, =3
        RET
        END
`
	testFile := "/tmp/dap_test_breakpoint.cas"
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	go func() {
		StartDAPServer(4714)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:4714")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Initialize
	initReq := map[string]interface{}{
		"seq":       1,
		"type":      "request",
		"command":   "initialize",
		"arguments": map[string]interface{}{},
	}
	sendDAPMessage(conn, initReq)
	readDAPMessage(reader)
	readDAPMessage(reader)

	// Launch
	launchReq := map[string]interface{}{
		"seq":     2,
		"type":    "request",
		"command": "launch",
		"arguments": map[string]interface{}{
			"program":     testFile,
			"stopOnEntry": false,
		},
	}
	sendDAPMessage(conn, launchReq)
	readDAPMessage(reader)

	// Set breakpoint
	bpReq := map[string]interface{}{
		"seq":     3,
		"type":    "request",
		"command": "setBreakpoints",
		"arguments": map[string]interface{}{
			"source": map[string]interface{}{
				"path": testFile,
			},
			"breakpoints": []interface{}{
				map[string]interface{}{
					"line": 5, // Line with ADDA instruction
				},
			},
		},
	}
	sendDAPMessage(conn, bpReq)
	resp, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read setBreakpoints response: %v", err)
	}

	if !resp["success"].(bool) {
		t.Errorf("setBreakpoints failed")
	}
}

// TestDAPVariables tests variable inspection
func TestDAPVariables(t *testing.T) {
	testProgram := `
MAIN    START
        LD      GR0, =42
        RET
        END
`
	testFile := "/tmp/dap_test_variables.cas"
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	go func() {
		StartDAPServer(4715)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:4715")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Initialize
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 1, "type": "request", "command": "initialize", "arguments": map[string]interface{}{},
	})
	readDAPMessage(reader)
	readDAPMessage(reader)

	// Launch with stopOnEntry
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 2, "type": "request", "command": "launch",
		"arguments": map[string]interface{}{"program": testFile, "stopOnEntry": true},
	})
	readDAPMessage(reader)

	// Configuration done
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 3, "type": "request", "command": "configurationDone", "arguments": map[string]interface{}{},
	})
	readDAPMessage(reader)
	readDAPMessage(reader) // Stopped event

	// Request stack trace
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 4, "type": "request", "command": "stackTrace",
		"arguments": map[string]interface{}{"threadId": 1},
	})
	_, _ = readDAPMessage(reader)

	// Request scopes
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 5, "type": "request", "command": "scopes",
		"arguments": map[string]interface{}{"frameId": 1},
	})
	scopesResp, _ := readDAPMessage(reader)

	if !scopesResp["success"].(bool) {
		t.Errorf("scopes request failed")
	}

	// Request variables
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 6, "type": "request", "command": "variables",
		"arguments": map[string]interface{}{"variablesReference": 1},
	})
	varsResp, _ := readDAPMessage(reader)

	if !varsResp["success"].(bool) {
		t.Errorf("variables request failed")
	}

	body := varsResp["body"].(map[string]interface{})
	variables := body["variables"].([]interface{})

	if len(variables) < 11 { // PC, FR, GR0-GR7, SP
		t.Errorf("Expected at least 11 variables, got %d", len(variables))
	}
}

// TestDAPDisconnect tests disconnection
func TestDAPDisconnect(t *testing.T) {
	go func() {
		StartDAPServer(4716)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:4716")
	if err != nil {
		t.Fatalf("Failed to connect to DAP server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Initialize
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 1, "type": "request", "command": "initialize", "arguments": map[string]interface{}{},
	})
	readDAPMessage(reader)
	readDAPMessage(reader)

	// Disconnect
	sendDAPMessage(conn, map[string]interface{}{
		"seq": 2, "type": "request", "command": "disconnect", "arguments": map[string]interface{}{},
	})
	
	resp, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read disconnect response: %v", err)
	}

	if !resp["success"].(bool) {
		t.Errorf("disconnect failed")
	}

	// Should receive terminated event
	evt, err := readDAPMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read terminated event: %v", err)
	}

	if evt["event"] != "terminated" {
		t.Errorf("Expected terminated event, got %v", evt["event"])
	}
}

// Helper function to send DAP message
func sendDAPMessage(conn net.Conn, msg map[string]interface{}) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	_, err = conn.Write([]byte(header))
	if err != nil {
		return err
	}

	_, err = conn.Write(content)
	return err
}

// Helper function to read DAP message
func readDAPMessage(reader *bufio.Reader) (map[string]interface{}, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
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
	n := 0
	for n < contentLength {
		read, err := reader.Read(content[n:])
		if err != nil {
			return nil, err
		}
		n += read
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, err
	}

	return msg, nil
}
