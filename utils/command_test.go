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

package utils

import (
	"runtime"
	"strings"
	"testing"
)

func TestCommandExists_ExistingCommand(t *testing.T) {
	// "echo" should exist on all platforms
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
	} else {
		cmd = "sh"
	}

	if !CommandExists(cmd) {
		t.Errorf("Expected %s to exist", cmd)
	}
}

func TestCommandExists_NonExistingCommand(t *testing.T) {
	if CommandExists("this-command-definitely-does-not-exist-12345") {
		t.Error("Expected non-existing command to return false")
	}
}

func TestRunCommand_Success(t *testing.T) {
	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "hello"}
	} else {
		cmd = "echo"
		args = []string{"hello"}
	}

	output, err := RunCommand(cmd, args...)
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !strings.Contains(string(output), "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", string(output))
	}
}

func TestRunCommand_Failure(t *testing.T) {
	_, err := RunCommand("this-command-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("Expected RunCommand to fail for non-existing command")
	}
}

func TestRunCommandSilent_Success(t *testing.T) {
	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "hello"}
	} else {
		cmd = "echo"
		args = []string{"hello"}
	}

	err := RunCommandSilent(cmd, args...)
	if err != nil {
		t.Fatalf("RunCommandSilent failed: %v", err)
	}
}

func TestRunCommandSilent_Failure(t *testing.T) {
	err := RunCommandSilent("this-command-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("Expected RunCommandSilent to fail for non-existing command")
	}
}
