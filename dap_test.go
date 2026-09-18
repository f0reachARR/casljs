package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDAPDebugSession(t *testing.T) {
	source := filepath.Join(t.TempDir(), "debug.cas")
	program := "MAIN START\n\tLAD GR1,1\n\tLAD GR2,2\n\tRET\n\tEND\n"
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}

	asm := newAssemblerState()
	binary, startLabel, err := assemble(source, asm)
	if err != nil {
		t.Fatal(err)
	}
	memory := make([]uint16, 0x10000)
	copy(memory, binary)
	machineState := []int{expandLabel(asm.symtbl, startLabel), FR_PLUS, 0, 0, 0, 0, 0, 0, 0, 0, STACK_TOP}

	server, client := net.Pipe()
	defer client.Close()
	adapter := newDAPAdapter(server, server, source, asm, memory, machineState)
	done := make(chan error, 1)
	go func() {
		_, err := adapter.serve()
		done <- err
	}()

	reader := bufio.NewReader(client)
	sendDAPRequest(t, client, 1, "initialize", map[string]interface{}{})
	requireDAPMessage(t, reader, "response", "initialize")

	sendDAPRequest(t, client, 2, "launch", map[string]interface{}{})
	requireDAPMessage(t, reader, "response", "launch")
	requireDAPMessage(t, reader, "event", "initialized")

	sendDAPRequest(t, client, 3, "setBreakpoints", map[string]interface{}{
		"source":      map[string]string{"path": source},
		"breakpoints": []map[string]int{{"line": 3}},
	})
	breakpointResponse := requireDAPMessage(t, reader, "response", "setBreakpoints")
	breakpoints := breakpointResponse["body"].(map[string]interface{})["breakpoints"].([]interface{})
	if !breakpoints[0].(map[string]interface{})["verified"].(bool) {
		t.Fatal("expected breakpoint to be verified")
	}

	sendDAPRequest(t, client, 4, "configurationDone", map[string]interface{}{})
	requireDAPMessage(t, reader, "response", "configurationDone")
	requireDAPMessage(t, reader, "event", "stopped")

	sendDAPRequest(t, client, 5, "continue", map[string]interface{}{"threadId": dapThreadID})
	requireDAPMessage(t, reader, "response", "continue")
	stopped := requireDAPMessage(t, reader, "event", "stopped")
	if reason := stopped["body"].(map[string]interface{})["reason"]; reason != "breakpoint" {
		t.Fatalf("stopped reason = %v, want breakpoint", reason)
	}
	if machineState[PC] != 2 {
		t.Fatalf("PC = %d, want 2", machineState[PC])
	}

	sendDAPRequest(t, client, 6, "stackTrace", map[string]interface{}{"threadId": dapThreadID})
	stack := requireDAPMessage(t, reader, "response", "stackTrace")
	frame := stack["body"].(map[string]interface{})["stackFrames"].([]interface{})[0].(map[string]interface{})
	if line := int(frame["line"].(float64)); line != 3 {
		t.Fatalf("stack frame line = %d, want 3", line)
	}

	sendDAPRequest(t, client, 7, "variables", map[string]interface{}{"variablesReference": 1})
	variables := requireDAPMessage(t, reader, "response", "variables")
	registers := variables["body"].(map[string]interface{})["variables"].([]interface{})
	if len(registers) != 11 {
		t.Fatalf("register count = %d, want 11", len(registers))
	}

	sendDAPRequest(t, client, 8, "next", map[string]interface{}{"threadId": dapThreadID})
	requireDAPMessage(t, reader, "response", "next")
	requireDAPMessage(t, reader, "event", "stopped")
	if machineState[PC] != 4 {
		t.Fatalf("PC = %d, want 4", machineState[PC])
	}

	sendDAPRequest(t, client, 9, "next", map[string]interface{}{"threadId": dapThreadID})
	requireDAPMessage(t, reader, "response", "next")
	requireDAPMessage(t, reader, "event", "terminated")

	sendDAPRequest(t, client, 10, "disconnect", map[string]interface{}{})
	requireDAPMessage(t, reader, "response", "disconnect")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDAPBreakpointWithoutCodeIsMoved(t *testing.T) {
	source := filepath.Join(t.TempDir(), "debug.cas")
	if err := os.WriteFile(source, []byte("MAIN START\n; comment\n\tNOP\n\tRET\n\tEND\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	asm := newAssemblerState()
	binary, startLabel, err := assemble(source, asm)
	if err != nil {
		t.Fatal(err)
	}
	memory := make([]uint16, 0x10000)
	copy(memory, binary)
	machineState := []int{expandLabel(asm.symtbl, startLabel), FR_PLUS, 0, 0, 0, 0, 0, 0, 0, 0, STACK_TOP}
	var output net.Conn
	server, client := net.Pipe()
	output = client
	defer output.Close()
	adapter := newDAPAdapter(server, server, source, asm, memory, machineState)
	defer server.Close()

	done := make(chan error, 1)
	go func() {
		request, readErr := adapter.protocol.readRequest()
		if readErr != nil {
			done <- readErr
			return
		}
		_, handleErr := adapter.handle(request)
		done <- handleErr
	}()
	sendDAPRequest(t, output, 1, "setBreakpoints", map[string]interface{}{
		"source":      map[string]string{"path": source},
		"breakpoints": []map[string]int{{"line": 2}},
	})
	response := requireDAPMessage(t, bufio.NewReader(output), "response", "setBreakpoints")
	result := response["body"].(map[string]interface{})["breakpoints"].([]interface{})[0].(map[string]interface{})
	if line := int(result["line"].(float64)); line != 3 {
		t.Fatalf("breakpoint line = %d, want 3", line)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDAPPauseStopsRunningProgram(t *testing.T) {
	source := filepath.Join(t.TempDir(), "loop.cas")
	if err := os.WriteFile(source, []byte("MAIN START\n\tJUMP MAIN\n\tEND\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	asm := newAssemblerState()
	binary, startLabel, err := assemble(source, asm)
	if err != nil {
		t.Fatal(err)
	}
	memory := make([]uint16, 0x10000)
	copy(memory, binary)
	machineState := []int{expandLabel(asm.symtbl, startLabel), FR_PLUS, 0, 0, 0, 0, 0, 0, 0, 0, STACK_TOP}

	server, client := net.Pipe()
	defer client.Close()
	adapter := newDAPAdapter(server, server, source, asm, memory, machineState)
	done := make(chan error, 1)
	go func() {
		_, err := adapter.serve()
		done <- err
	}()
	reader := bufio.NewReader(client)

	sendDAPRequest(t, client, 1, "continue", map[string]interface{}{"threadId": dapThreadID})
	requireDAPMessage(t, reader, "response", "continue")
	sendDAPRequest(t, client, 2, "continue", map[string]interface{}{"threadId": dapThreadID})
	alreadyRunning := requireDAPMessage(t, reader, "response", "continue")
	if alreadyRunning["success"].(bool) {
		t.Fatal("second continue unexpectedly succeeded")
	}
	sendDAPRequest(t, client, 3, "setBreakpoints", map[string]interface{}{
		"source":      map[string]string{"path": source + ".other"},
		"breakpoints": []map[string]int{{"line": 2}},
	})
	requireDAPMessage(t, reader, "response", "setBreakpoints")
	sendDAPRequest(t, client, 4, "pause", map[string]interface{}{"threadId": dapThreadID})
	requireDAPMessage(t, reader, "response", "pause")
	stopped := requireDAPMessage(t, reader, "event", "stopped")
	if reason := stopped["body"].(map[string]interface{})["reason"]; reason != "pause" {
		t.Fatalf("stopped reason = %v, want pause", reason)
	}

	sendDAPRequest(t, client, 5, "disconnect", map[string]interface{}{})
	requireDAPMessage(t, reader, "response", "disconnect")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func sendDAPRequest(t *testing.T, writer io.Writer, sequence int, command string, arguments interface{}) {
	t.Helper()
	content, err := json.Marshal(map[string]interface{}{
		"seq": sequence, "type": "request", "command": command, "arguments": arguments,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n%s", len(content), content); err != nil {
		t.Fatal(err)
	}
}

func requireDAPMessage(t *testing.T, reader *bufio.Reader, messageType, name string) map[string]interface{} {
	t.Helper()
	protocol := dapProtocol{reader: reader}
	contentLength := -1
	for {
		line, err := protocol.reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
		if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength); err != nil {
			t.Fatal(err)
		}
	}
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(protocol.reader, content); err != nil {
		t.Fatal(err)
	}
	var message map[string]interface{}
	if err := json.Unmarshal(content, &message); err != nil {
		t.Fatal(err)
	}
	if message["type"] != messageType {
		t.Fatalf("message type = %v, want %s", message["type"], messageType)
	}
	key := "command"
	if messageType == "event" {
		key = "event"
	}
	if message[key] != name {
		t.Fatalf("%s = %v, want %s", key, message[key], name)
	}
	return message
}
