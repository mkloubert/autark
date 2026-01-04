// The MIT License (MIT)
// Copyright (c) 2026 Marcel Joachim Kloubert <https://marcel.coffee>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mkloubert/autark/app"
)

func TestDoctorCommand_Exists(t *testing.T) {
	a, err := app.NewAppContext()
	if err != nil {
		t.Fatalf("Failed to create app context: %v", err)
	}

	InitCommands(a)

	rootCmd := a.RootCommand()
	doctorCmd, _, err := rootCmd.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("Doctor command not found: %v", err)
	}

	if doctorCmd == nil {
		t.Fatal("Doctor command is nil")
	}

	if doctorCmd.Use != "doctor" {
		t.Errorf("Expected command 'doctor', got '%s'", doctorCmd.Use)
	}
}

func TestDoctorCommand_Aliases(t *testing.T) {
	a, err := app.NewAppContext()
	if err != nil {
		t.Fatalf("Failed to create app context: %v", err)
	}

	InitCommands(a)

	rootCmd := a.RootCommand()

	// Test "doc" alias
	docCmd, _, err := rootCmd.Find([]string{"doc"})
	if err != nil {
		t.Fatalf("Doc alias not found: %v", err)
	}
	if docCmd == nil {
		t.Fatal("Doc alias is nil")
	}

	// Test "d" alias
	dCmd, _, err := rootCmd.Find([]string{"d"})
	if err != nil {
		t.Fatalf("D alias not found: %v", err)
	}
	if dCmd == nil {
		t.Fatal("D alias is nil")
	}
}

func TestDoctorCommand_RepairFlag(t *testing.T) {
	a, err := app.NewAppContext()
	if err != nil {
		t.Fatalf("Failed to create app context: %v", err)
	}

	InitCommands(a)

	rootCmd := a.RootCommand()
	doctorCmd, _, err := rootCmd.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("Doctor command not found: %v", err)
	}

	repairFlag := doctorCmd.Flags().Lookup("repair")
	if repairFlag == nil {
		t.Fatal("Repair flag not found")
	}

	if repairFlag.Shorthand != "r" {
		t.Errorf("Expected shorthand 'r', got '%s'", repairFlag.Shorthand)
	}

	if repairFlag.DefValue != "false" {
		t.Errorf("Expected default value 'false', got '%s'", repairFlag.DefValue)
	}
}

func TestCheckGit(t *testing.T) {
	result := checkGit()

	if result == nil {
		t.Fatal("checkGit returned nil")
	}

	if result.Name != "git" {
		t.Errorf("Expected name 'git', got '%s'", result.Name)
	}

	// Git should be installed in the test environment
	if result.Installed {
		if result.Version == "" {
			t.Error("Git is installed but version is empty")
		}
		if !strings.Contains(result.Version, "git version") {
			t.Errorf("Unexpected git version format: %s", result.Version)
		}
	}
}

func TestCheckDocker(t *testing.T) {
	result := checkDocker()

	if result == nil {
		t.Fatal("checkDocker returned nil")
	}

	if result.Name != "docker" {
		t.Errorf("Expected name 'docker', got '%s'", result.Name)
	}

	// Note: Docker may or may not be installed in the test environment
	// We just verify the function returns a valid result
	if result.Installed && result.Version == "" {
		t.Error("Docker is installed but version is empty")
	}
}

func TestDoctorResult_PrintResult(t *testing.T) {
	a, err := app.NewAppContext()
	if err != nil {
		t.Fatalf("Failed to create app context: %v", err)
	}

	// Test with installed tool
	result := &DoctorResult{
		Name:      "test-tool",
		Installed: true,
		Version:   "v1.0.0",
	}

	// Capture stdout
	var buf bytes.Buffer
	stdout := a.Stdout()
	if stdout != nil {
		// Note: In a real test we'd redirect stdout
		// For now we just verify the function doesn't panic
		printResult(a, result)
	}
	_ = buf // silence unused variable warning

	// Test with uninstalled tool
	result2 := &DoctorResult{
		Name:      "missing-tool",
		Installed: false,
	}
	printResult(a, result2)
}

func TestDoctorOptions(t *testing.T) {
	opts := &DoctorOptions{
		Repair: false,
	}

	if opts.Repair != false {
		t.Error("Expected Repair to be false by default")
	}

	opts.Repair = true
	if opts.Repair != true {
		t.Error("Failed to set Repair to true")
	}
}
