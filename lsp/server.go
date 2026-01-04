package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// Server represents the LSP server for CASL2
type Server struct {
	documents map[string]*Document
	logger    *log.Logger
	writer    io.Writer
}

// Document represents an open text document
type Document struct {
	URI     string
	Content string
	Version int
}

// NewServer creates a new LSP server
func NewServer() *Server {
	logger := log.New(os.Stderr, "[CASL2-LSP] ", log.LstdFlags)
	return &Server{
		documents: make(map[string]*Document),
		logger:    logger,
	}
}

// Start starts the LSP server
func (s *Server) Start(reader io.Reader, writer io.Writer) error {
	s.writer = writer
	s.logger.Println("CASL2 Language Server starting...")
	
	scanner := bufio.NewScanner(reader)
	scanner.Split(s.scanLSPMessages)
	
	for scanner.Scan() {
		messageBytes := scanner.Bytes()
		
		var msg map[string]interface{}
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			s.logger.Printf("Error decoding message: %v", err)
			continue
		}
		
		s.logger.Printf("Received message: method=%v", msg["method"])
		
		// Handle the message
		response := s.handleMessage(msg)
		if response != nil {
			s.sendResponse(response)
		}
	}
	
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	
	return nil
}

// scanLSPMessages is a split function for bufio.Scanner that returns LSP messages
func (s *Server) scanLSPMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	
	// Look for Content-Length header
	headerEnd := -1
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			headerEnd = i + 4
			break
		}
	}
	
	if headerEnd == -1 {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete message")
		}
		return 0, nil, nil
	}
	
	// Parse Content-Length
	headers := string(data[:headerEnd])
	contentLength := 0
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(line, "Content-Length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				contentLength, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
	}
	
	if contentLength == 0 {
		return headerEnd, nil, nil
	}
	
	messageEnd := headerEnd + contentLength
	if len(data) < messageEnd {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete message body")
		}
		return 0, nil, nil
	}
	
	return messageEnd, data[headerEnd:messageEnd], nil
}

// sendResponse sends a response to the client
func (s *Server) sendResponse(response map[string]interface{}) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		s.logger.Printf("Error marshaling response: %v", err)
		return
	}
	
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(responseJSON))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		s.logger.Printf("Error writing header: %v", err)
		return
	}
	if _, err := s.writer.Write(responseJSON); err != nil {
		s.logger.Printf("Error writing response: %v", err)
		return
	}
}

// handleMessage handles an incoming LSP message
func (s *Server) handleMessage(msg map[string]interface{}) map[string]interface{} {
	method, ok := msg["method"].(string)
	if !ok {
		return nil
	}
	
	id, hasID := msg["id"]
	params, _ := msg["params"].(map[string]interface{})
	
	s.logger.Printf("Handling method: %s", method)
	
	switch method {
	case "initialize":
		if hasID {
			return s.handleInitialize(id, params)
		}
	case "initialized":
		return nil
	case "textDocument/didOpen":
		s.handleDidOpen(params)
		return nil
	case "textDocument/didChange":
		s.handleDidChange(params)
		return nil
	case "textDocument/didClose":
		s.handleDidClose(params)
		return nil
	case "textDocument/hover":
		if hasID {
			return s.handleHover(id, params)
		}
	case "textDocument/completion":
		if hasID {
			return s.handleCompletion(id, params)
		}
	case "textDocument/definition":
		if hasID {
			return s.handleDefinition(id, params)
		}
	case "textDocument/documentSymbol":
		if hasID {
			return s.handleDocumentSymbol(id, params)
		}
	case "shutdown":
		if hasID {
			return s.handleShutdown(id)
		}
	case "exit":
		os.Exit(0)
		return nil
	default:
		s.logger.Printf("Unhandled method: %s", method)
		return nil
	}
	return nil
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(id interface{}, params map[string]interface{}) map[string]interface{} {
	capabilities := map[string]interface{}{
		"textDocumentSync": map[string]interface{}{
			"openClose": true,
			"change":    1, // Full document sync
		},
		"hoverProvider":           true,
		"completionProvider":      map[string]interface{}{},
		"definitionProvider":      true,
		"documentSymbolProvider":  true,
	}
	
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"capabilities": capabilities,
			"serverInfo": map[string]interface{}{
				"name":    "CASL2 Language Server",
				"version": "1.0.0",
			},
		},
	}
}

// handleDidOpen handles the textDocument/didOpen notification
func (s *Server) handleDidOpen(params map[string]interface{}) {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return
	}
	
	uri, _ := textDoc["uri"].(string)
	text, _ := textDoc["text"].(string)
	version, _ := textDoc["version"].(float64)
	
	s.documents[uri] = &Document{
		URI:     uri,
		Content: text,
		Version: int(version),
	}
	
	s.logger.Printf("Document opened: %s", uri)
	s.publishDiagnostics(uri, text)
}

// handleDidChange handles the textDocument/didChange notification
func (s *Server) handleDidChange(params map[string]interface{}) {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return
	}
	
	uri, _ := textDoc["uri"].(string)
	changes, _ := params["contentChanges"].([]interface{})
	
	if len(changes) > 0 {
		change, ok := changes[0].(map[string]interface{})
		if ok {
			text, _ := change["text"].(string)
			if doc, exists := s.documents[uri]; exists {
				doc.Content = text
				s.publishDiagnostics(uri, text)
			}
		}
	}
}

// handleDidClose handles the textDocument/didClose notification
func (s *Server) handleDidClose(params map[string]interface{}) {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return
	}
	
	uri, _ := textDoc["uri"].(string)
	delete(s.documents, uri)
	s.logger.Printf("Document closed: %s", uri)
}

// handleHover handles the textDocument/hover request
func (s *Server) handleHover(id interface{}, params map[string]interface{}) map[string]interface{} {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return s.nullResponse(id)
	}
	
	uri, _ := textDoc["uri"].(string)
	position, _ := params["position"].(map[string]interface{})
	line, _ := position["line"].(float64)
	character, _ := position["character"].(float64)
	
	doc, exists := s.documents[uri]
	if !exists {
		return s.nullResponse(id)
	}
	
	word := s.getWordAtPosition(doc.Content, int(line), int(character))
	hover := s.getHoverInfo(word)
	
	if hover == "" {
		return s.nullResponse(id)
	}
	
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"contents": hover,
		},
	}
}

// handleCompletion handles the textDocument/completion request
func (s *Server) handleCompletion(id interface{}, params map[string]interface{}) map[string]interface{} {
	completions := s.getCompletions()
	
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  completions,
	}
}

// handleDefinition handles the textDocument/definition request
func (s *Server) handleDefinition(id interface{}, params map[string]interface{}) map[string]interface{} {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return s.nullResponse(id)
	}
	
	uri, _ := textDoc["uri"].(string)
	position, _ := params["position"].(map[string]interface{})
	line, _ := position["line"].(float64)
	character, _ := position["character"].(float64)
	
	doc, exists := s.documents[uri]
	if !exists {
		return s.nullResponse(id)
	}
	
	word := s.getWordAtPosition(doc.Content, int(line), int(character))
	location := s.findLabelDefinition(doc.Content, word, uri)
	
	if location == nil {
		return s.nullResponse(id)
	}
	
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  location,
	}
}

// handleDocumentSymbol handles the textDocument/documentSymbol request
func (s *Server) handleDocumentSymbol(id interface{}, params map[string]interface{}) map[string]interface{} {
	textDoc, ok := params["textDocument"].(map[string]interface{})
	if !ok {
		return s.nullResponse(id)
	}
	
	uri, _ := textDoc["uri"].(string)
	doc, exists := s.documents[uri]
	if !exists {
		return s.nullResponse(id)
	}
	
	symbols := s.getDocumentSymbols(doc.Content, uri)
	
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  symbols,
	}
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(id interface{}) map[string]interface{} {
	s.logger.Println("Shutdown requested")
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  nil,
	}
}

// nullResponse returns a null response
func (s *Server) nullResponse(id interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  nil,
	}
}

// publishDiagnostics publishes diagnostics for a document
func (s *Server) publishDiagnostics(uri, text string) {
	diagnostics := s.analyzeSyntax(text)
	
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params": map[string]interface{}{
			"uri":         uri,
			"diagnostics": diagnostics,
		},
	}
	
	s.sendResponse(notification)
}

// analyzeSyntax analyzes the syntax of CASL2 code
func (s *Server) analyzeSyntax(text string) []map[string]interface{} {
	diagnostics := []map[string]interface{}{}
	lines := strings.Split(text, "\n")
	
	knownInstructions := []string{
		"NOP", "LD", "ST", "LAD", "ADDA", "SUBA", "ADDL", "SUBL",
		"MULA", "DIVA", "MULL", "DIVL", "AND", "OR", "XOR",
		"CPA", "CPL", "SLA", "SRA", "SLL", "SRL",
		"JMI", "JNZ", "JZE", "JUMP", "JPL", "JOV",
		"PUSH", "POP", "CALL", "RET", "SVC",
		"START", "END", "DS", "DC", "IN", "OUT", "RPUSH", "RPOP",
	}
	
	instructionSet := make(map[string]bool)
	for _, inst := range knownInstructions {
		instructionSet[inst] = true
	}
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		
		// Remove comments
		if idx := strings.Index(line, ";"); idx >= 0 {
			line = line[:idx]
		}
		
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		
		// Check if line starts with whitespace (instruction without label)
		startsWithWhitespace := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
		
		var instruction string
		if startsWithWhitespace {
			instruction = fields[0]
		} else if len(fields) > 1 {
			instruction = fields[1]
		} else {
			// Only label, no instruction
			continue
		}
		
		// Check if instruction is valid
		if !instructionSet[strings.ToUpper(instruction)] {
			diagnostics = append(diagnostics, map[string]interface{}{
				"range": map[string]interface{}{
					"start": map[string]interface{}{"line": i, "character": 0},
					"end":   map[string]interface{}{"line": i, "character": len(line)},
				},
				"severity": 1, // Error
				"message":  fmt.Sprintf("Unknown instruction: %s", instruction),
				"source":   "casl2",
			})
		}
	}
	
	return diagnostics
}

// getWordAtPosition gets the word at a specific position
func (s *Server) getWordAtPosition(text string, line, character int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	
	currentLine := lines[line]
	if character < 0 || character > len(currentLine) {
		return ""
	}
	
	// Find word boundaries
	start := character
	for start > 0 && isWordChar(currentLine[start-1]) {
		start--
	}
	
	end := character
	for end < len(currentLine) && isWordChar(currentLine[end]) {
		end++
	}
	
	return currentLine[start:end]
}

// isWordChar checks if a character is part of a word
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || 
	       (c >= '0' && c <= '9') || c == '_' || c == '$' || c == '%' || c == '.'
}

// getHoverInfo returns hover information for a word
func (s *Server) getHoverInfo(word string) string {
	instructionInfo := map[string]string{
		"NOP":   "No operation",
		"LD":    "Load - LD r,adr,x or LD r1,r2",
		"ST":    "Store - ST r,adr,x",
		"LAD":   "Load address - LAD r,adr,x",
		"ADDA":  "Add arithmetic - ADDA r,adr,x or ADDA r1,r2",
		"SUBA":  "Subtract arithmetic - SUBA r,adr,x or SUBA r1,r2",
		"ADDL":  "Add logical - ADDL r,adr,x or ADDL r1,r2",
		"SUBL":  "Subtract logical - SUBL r,adr,x or SUBL r1,r2",
		"MULA":  "Multiply arithmetic - MULA r,adr,x or MULA r1,r2",
		"DIVA":  "Divide arithmetic - DIVA r,adr,x or DIVA r1,r2",
		"MULL":  "Multiply logical - MULL r,adr,x or MULL r1,r2",
		"DIVL":  "Divide logical - DIVL r,adr,x or DIVL r1,r2",
		"AND":   "Logical AND - AND r,adr,x or AND r1,r2",
		"OR":    "Logical OR - OR r,adr,x or OR r1,r2",
		"XOR":   "Logical XOR - XOR r,adr,x or XOR r1,r2",
		"CPA":   "Compare arithmetic - CPA r,adr,x or CPA r1,r2",
		"CPL":   "Compare logical - CPL r,adr,x or CPL r1,r2",
		"SLA":   "Shift left arithmetic - SLA r,adr,x",
		"SRA":   "Shift right arithmetic - SRA r,adr,x",
		"SLL":   "Shift left logical - SLL r,adr,x",
		"SRL":   "Shift right logical - SRL r,adr,x",
		"JMI":   "Jump if minus - JMI adr,x",
		"JNZ":   "Jump if not zero - JNZ adr,x",
		"JZE":   "Jump if zero - JZE adr,x",
		"JUMP":  "Unconditional jump - JUMP adr,x",
		"JPL":   "Jump if plus - JPL adr,x",
		"JOV":   "Jump if overflow - JOV adr,x",
		"PUSH":  "Push to stack - PUSH adr,x",
		"POP":   "Pop from stack - POP r",
		"CALL":  "Call subroutine - CALL adr,x",
		"RET":   "Return from subroutine",
		"SVC":   "Supervisor call - SVC adr,x",
		"START": "Start of program - label START [address]",
		"END":   "End of program",
		"DS":    "Define storage - DS n",
		"DC":    "Define constant - DC value or DC 'string'",
		"IN":    "Input - IN ibuf,ilen",
		"OUT":   "Output - OUT obuf,olen",
		"RPUSH": "Register push - RPUSH",
		"RPOP":  "Register pop - RPOP",
	}
	
	upperWord := strings.ToUpper(word)
	if info, ok := instructionInfo[upperWord]; ok {
		return info
	}
	
	// Check for registers
	if strings.HasPrefix(upperWord, "GR") && len(word) == 3 {
		digit := word[2]
		if digit >= '0' && digit <= '7' {
			return fmt.Sprintf("General Register %c", digit)
		}
	}
	
	return ""
}

// getCompletions returns completion items
func (s *Server) getCompletions() []map[string]interface{} {
	completions := []map[string]interface{}{}
	
	instructions := []string{
		"NOP", "LD", "ST", "LAD", "ADDA", "SUBA", "ADDL", "SUBL",
		"MULA", "DIVA", "MULL", "DIVL", "AND", "OR", "XOR",
		"CPA", "CPL", "SLA", "SRA", "SLL", "SRL",
		"JMI", "JNZ", "JZE", "JUMP", "JPL", "JOV",
		"PUSH", "POP", "CALL", "RET", "SVC",
		"START", "END", "DS", "DC", "IN", "OUT", "RPUSH", "RPOP",
	}
	
	for _, inst := range instructions {
		completions = append(completions, map[string]interface{}{
			"label": inst,
			"kind":  3, // Function
		})
	}
	
	// Add registers
	for i := 0; i <= 7; i++ {
		reg := fmt.Sprintf("GR%d", i)
		completions = append(completions, map[string]interface{}{
			"label": reg,
			"kind":  5, // Variable
		})
	}
	
	return completions
}

// findLabelDefinition finds the definition of a label
func (s *Server) findLabelDefinition(text, label, uri string) map[string]interface{} {
	lines := strings.Split(text, "\n")
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		
		// Remove comments
		if idx := strings.Index(line, ";"); idx >= 0 {
			line = line[:idx]
		}
		
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		
		// Check if line starts with a label (no leading whitespace)
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			// First field is a label
			if strings.EqualFold(fields[0], label) {
				return map[string]interface{}{
					"uri": uri,
					"range": map[string]interface{}{
						"start": map[string]interface{}{"line": i, "character": 0},
						"end":   map[string]interface{}{"line": i, "character": len(fields[0])},
					},
				}
			}
		}
	}
	
	return nil
}

// getDocumentSymbols returns document symbols
func (s *Server) getDocumentSymbols(text, uri string) []map[string]interface{} {
	symbols := []map[string]interface{}{}
	lines := strings.Split(text, "\n")
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		
		// Remove comments
		if idx := strings.Index(line, ";"); idx >= 0 {
			line = line[:idx]
		}
		
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		
		// Check if line starts with a label (no leading whitespace)
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			label := fields[0]
			symbols = append(symbols, map[string]interface{}{
				"name": label,
				"kind": 12, // Constant
				"location": map[string]interface{}{
					"uri": uri,
					"range": map[string]interface{}{
						"start": map[string]interface{}{"line": i, "character": 0},
						"end":   map[string]interface{}{"line": i, "character": len(label)},
					},
				},
			})
		}
	}
	
	return symbols
}
