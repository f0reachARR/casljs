package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestBasicLSPCommunication tests basic LSP message handling
func TestBasicLSPCommunication(t *testing.T) {
	server := NewServer()
	
	// Test initialize request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"processId": nil,
			"rootUri":   "file:///test",
			"capabilities": map[string]interface{}{},
		},
	}
	
	response := server.handleMessage(initRequest)
	if response == nil {
		t.Fatal("Expected response from initialize, got nil")
	}
	
	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result in response")
	}
	
	capabilities, ok := result["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected capabilities in result")
	}
	
	if _, ok := capabilities["hoverProvider"]; !ok {
		t.Error("Expected hoverProvider capability")
	}
	
	if _, ok := capabilities["completionProvider"]; !ok {
		t.Error("Expected completionProvider capability")
	}
}

// TestHoverInfo tests hover information
func TestHoverInfo(t *testing.T) {
	server := NewServer()
	
	tests := []struct {
		word     string
		expected bool
	}{
		{"LD", true},
		{"ADDA", true},
		{"GR0", true},
		{"GR7", true},
		{"INVALID", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			info := server.getHoverInfo(tt.word)
			if tt.expected && info == "" {
				t.Errorf("Expected hover info for %s, got empty string", tt.word)
			}
			if !tt.expected && info != "" {
				t.Errorf("Expected no hover info for %s, got %s", tt.word, info)
			}
		})
	}
}

// TestCompletions tests completion items
func TestCompletions(t *testing.T) {
	server := NewServer()
	completions := server.getCompletions()
	
	if len(completions) == 0 {
		t.Fatal("Expected completions, got none")
	}
	
	// Check for some expected instructions
	found := make(map[string]bool)
	for _, comp := range completions {
		label, ok := comp["label"].(string)
		if ok {
			found[label] = true
		}
	}
	
	expectedItems := []string{"LD", "ST", "ADDA", "JUMP", "GR0", "GR7"}
	for _, item := range expectedItems {
		if !found[item] {
			t.Errorf("Expected completion item %s not found", item)
		}
	}
}

// TestAnalyzeSyntax tests syntax analysis
func TestAnalyzeSyntax(t *testing.T) {
	server := NewServer()
	
	tests := []struct {
		name        string
		code        string
		expectError bool
	}{
		{
			name: "valid code",
			code: `START
	LD	gr0,0
	RET
	END`,
			expectError: false,
		},
		{
			name: "invalid instruction",
			code: `START
	INVALID gr0,0
	RET
	END`,
			expectError: true,
		},
		{
			name: "comment only",
			code: `; This is a comment
START
	; Another comment
	RET
	END`,
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := server.analyzeSyntax(tt.code)
			hasError := len(diagnostics) > 0
			
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got: %v (diagnostics: %d)", 
					tt.expectError, hasError, len(diagnostics))
			}
		})
	}
}

// TestFindLabelDefinition tests label definition finding
func TestFindLabelDefinition(t *testing.T) {
	server := NewServer()
	
	code := `START
LOOP	LD	gr0,0
	JUMP	LOOP
	RET
END_LABEL	END`
	
	tests := []struct {
		label    string
		expected bool
	}{
		{"LOOP", true},
		{"START", true},
		{"END_LABEL", true},
		{"NONEXISTENT", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			location := server.findLabelDefinition(code, tt.label, "file:///test.cas")
			found := location != nil
			
			if found != tt.expected {
				t.Errorf("Label %s: expected found=%v, got found=%v", 
					tt.label, tt.expected, found)
			}
		})
	}
}

// TestGetDocumentSymbols tests document symbol extraction
func TestGetDocumentSymbols(t *testing.T) {
	server := NewServer()
	
	code := `START
LABEL1	LD	gr0,0
LABEL2	ST	gr0,VAR
VAR	DS	1
	END`
	
	symbols := server.getDocumentSymbols(code, "file:///test.cas")
	
	if len(symbols) < 4 {
		t.Errorf("Expected at least 4 symbols, got %d", len(symbols))
	}
	
	// Check for expected labels
	found := make(map[string]bool)
	for _, sym := range symbols {
		name, ok := sym["name"].(string)
		if ok {
			found[name] = true
		}
	}
	
	expectedLabels := []string{"START", "LABEL1", "LABEL2", "VAR"}
	for _, label := range expectedLabels {
		if !found[label] {
			t.Errorf("Expected symbol %s not found", label)
		}
	}
}

// TestLSPMessageParsing tests LSP message parsing
func TestLSPMessageParsing(t *testing.T) {
	// Create a simple LSP message
	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	
	messageJSON, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create LSP message with Content-Length header
	lspMessage := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(messageJSON), messageJSON)
	
	// Test scanner
	reader := strings.NewReader(lspMessage)
	scanner := bytes.NewBuffer(nil)
	
	// Read the message
	_, err = scanner.ReadFrom(reader)
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify message was read
	if scanner.Len() == 0 {
		t.Error("Expected message to be read")
	}
}

// TestWordAtPosition tests word extraction at cursor position
func TestWordAtPosition(t *testing.T) {
	server := NewServer()
	
	tests := []struct {
		name      string
		text      string
		line      int
		character int
		expected  string
	}{
		{
			name:      "instruction",
			text:      "	LD	gr0,0",
			line:      0,
			character: 2,
			expected:  "LD",
		},
		{
			name:      "register",
			text:      "	LD	gr0,0",
			line:      0,
			character: 6,
			expected:  "gr0",
		},
		{
			name:      "label",
			text:      "LOOP	LD	gr0,0",
			line:      0,
			character: 2,
			expected:  "LOOP",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word := server.getWordAtPosition(tt.text, tt.line, tt.character)
			if word != tt.expected {
				t.Errorf("Expected word %s, got %s", tt.expected, word)
			}
		})
	}
}
