package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Test input configuration
type TestInput map[string][]string

func TestC2C2Samples(t *testing.T) {
	// Read input.json
	inputData, err := ioutil.ReadFile("test/input.json")
	if err != nil {
		t.Fatalf("Failed to read input.json: %v", err)
	}

	var testInputs TestInput
	if err := json.Unmarshal(inputData, &testInputs); err != nil {
		t.Fatalf("Failed to parse input.json: %v", err)
	}

	// Find all .cas files in test/samples
	casFiles, err := filepath.Glob("test/samples/**/*.cas")
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	for _, casFile := range casFiles {
		t.Run(filepath.Base(casFile), func(t *testing.T) {
			testSample(t, casFile, testInputs)
		})
	}
}

func testSample(t *testing.T, casFile string, testInputs TestInput) {
	baseName := filepath.Base(casFile)
	expectFile := filepath.Join("test/test_expects", baseName+".out")

	// Check if expect file exists
	if _, err := os.Stat(expectFile); os.IsNotExist(err) {
		t.Skipf("No expectation file for %s", baseName)
		return
	}

	// Read expected output
	expectedBytes, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Fatalf("Failed to read expectation file: %v", err)
	}
	expected := string(expectedBytes)

	// Build command arguments
	args := []string{"-n", "-q", "-r", casFile}
	if inputs, ok := testInputs[baseName]; ok {
		args = append(args, inputs...)
	}

	// Execute c2c2
	cmd := exec.Command("./c2c2", args...)
	output, err := cmd.CombinedOutput()
	
	// Check for errors (but allow "Program finished" errors)
	if err != nil {
		if !strings.Contains(string(output), "Program finished") {
			t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
		}
	}

	actual := string(output)

	// Compare outputs
	if actual != expected {
		t.Errorf("Output mismatch for %s\nExpected:\n%s\nActual:\n%s", baseName, expected, actual)
		
		// Show diff
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")
		
		maxLines := len(expectedLines)
		if len(actualLines) > maxLines {
			maxLines = len(actualLines)
		}
		
		for i := 0; i < maxLines; i++ {
			expLine := ""
			actLine := ""
			if i < len(expectedLines) {
				expLine = expectedLines[i]
			}
			if i < len(actualLines) {
				actLine = actualLines[i]
			}
			
			if expLine != actLine {
				t.Logf("Line %d differs:\n  Expected: %q\n  Actual:   %q", i+1, expLine, actLine)
			}
		}
	}
}
