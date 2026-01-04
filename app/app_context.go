// The MIT License (MIT)
// Copyright (c) 2026 Marcel Joachim Kloubert <https://marcel.coffee>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the “Software”), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package app

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mkloubert/autark/utils"
	"github.com/spf13/cobra"
)

// AppContext handles the current application context
type AppContext struct {
	config   *AppConfig
	logger   *log.Logger
	platform *utils.PlatformInfo
	stderr   *os.File
	stdin    *os.File
	stdout   *os.File
	rootCmd  *cobra.Command
}

// NewAppContext creates a new instance of AppContext and returns
// an error on failure
func NewAppContext() (*AppContext, error) {
	a := &AppContext{}

	config, err := NewAppConfig()
	if err != nil {
		return nil, nil
	}

	rootCmd := &cobra.Command{
		Use:   "autark",
		Short: "Installs server software with Docker Compose",
		Long:  `A platform independent Command Line Tool that installs a server software stack with ease using Docker Compose.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	flags := rootCmd.PersistentFlags()
	flags.BoolVarP(&config.Verbose, "verbose", "", false, "verbose output")

	a.config = config
	a.platform = utils.DetectPlatform()
	a.rootCmd = rootCmd
	a.stderr = os.Stderr
	a.stdin = os.Stdin
	a.stdout = os.Stdout

	newLogger := log.Default()
	newLogger.SetPrefix("[autark] ")
	newLogger.SetOutput(a.stderr)
	newLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	a.logger = newLogger

	return a, nil
}

// Config returns the current configuration
// of this app
func (a *AppContext) Config() *AppConfig {
	return a.config
}

// D logs a debug message via the logger of this app
func (a *AppContext) D(format string, args ...any) {
	if !a.Config().Verbose {
		return
	}

	a.logWithPrefix("[DEBUG] ", format, args...)
}

// E logs an error message via the logger of this app
func (a *AppContext) E(format string, args ...any) {
	a.logWithPrefix("[ERROR] ", format, args...)
}

// I logs an information message via the logger of this app
func (a *AppContext) I(format string, args ...any) {
	a.logWithPrefix("[INFO] ", format, args...)
}

// L returns the logger used by this app
func (a *AppContext) L() *log.Logger {
	return a.logger
}

func (a *AppContext) logWithPrefix(prefix string, format string, args ...any) {
	l := a.L()
	if l == nil {
		return
	}

	l.Printf("%s%s%s", prefix, fmt.Sprintf(format, args...), a.Config().EOL)
}

// P logs a panic message and finally executes panic function
func (a *AppContext) P(format string, args ...any) {
	l := a.L()
	if l == nil {
		panic(fmt.Sprintf(format, args...))
	}

	l.Panicf("%s%s%s", "[PANIC] ", fmt.Sprintf(format, args...), a.Config().EOL)
}

// Platform returns the platform information
// of this app
func (a *AppContext) Platform() *utils.PlatformInfo {
	return a.platform
}

// PromptPort prompts the user for a port number with a suggested default
func (a *AppContext) PromptPort(prompt string, defaultPort int) int {
	reader := bufio.NewReader(a.Stdin())

	for {
		a.WriteF("%s [%d]: ", prompt, defaultPort)

		input, err := reader.ReadString('\n')
		if err != nil {
			return defaultPort
		}

		input = strings.TrimSpace(input)

		if input == "" {
			return defaultPort
		}

		port, err := strconv.Atoi(input)
		if err != nil {
			a.WriteErrLn("Invalid port number. Please enter a valid number.")
			continue
		}

		if port < 1 || port > 65535 {
			a.WriteErrLn("Port must be between 1 and 65535.")
			continue
		}

		return port
	}
}

// PromptYesNo prompts the user with a yes/no question and returns true for yes
func (a *AppContext) PromptYesNo(prompt string, defaultYes bool) bool {
	for {
		reader := bufio.NewReader(a.Stdin())

		hint := "[y/N]"
		if defaultYes {
			hint = "[Y/n]"
		}

		a.WriteF("%s %s: ", prompt, hint)

		input, err := reader.ReadString('\n')
		if err != nil {
			return defaultYes
		}

		switch strings.TrimSpace(strings.ToLower(input)) {
		case "n", "no", "y", "yes":
			return input == "y" || input == "yes"
		case "":
			return defaultYes
		}
	}
}

// RootCommand returns the unterlying root command
// of this app
func (a *AppContext) RootCommand() *cobra.Command {
	return a.rootCmd
}

// Run runs this app and returns an error on failure
func (a *AppContext) Run() error {
	return a.rootCmd.Execute()
}

// Stderr returns standard error used by this app
func (a *AppContext) Stderr() *os.File {
	return a.stderr
}

// Stdin returns standard input used by this app
func (a *AppContext) Stdin() *os.File {
	return a.stdin
}

// Stdout returns standard output used by this app
func (a *AppContext) Stdout() *os.File {
	return a.stdout
}

// W logs a warning message via the logger of this app
func (a *AppContext) W(format string, args ...any) {
	a.logWithPrefix("[WARN] ", format, args...)
}

// Write writes binary data to standard output
// of this app
func (a *AppContext) Write(b []byte) (int, error) {
	stdout := a.Stdout()
	if stdout == nil {
		return len(b), nil
	}

	return stdout.Write(b)
}

// WriteErr writes binary data to standard error
// of this app
func (a *AppContext) WriteErr(b []byte) (int, error) {
	stderr := a.Stderr()
	if stderr == nil {
		return len(b), nil
	}

	return stderr.Write(b)
}

// WriteErrF writes formatted string data to standard error
// of this app
func (a *AppContext) WriteErrF(format string, args ...any) *AppContext {
	return a.WriteErrString(
		fmt.Sprintf(format, args...),
	)
}

// WriteErrLn writes string data to standard error
// of this app and adds EOL
func (a *AppContext) WriteErrLn(s string) *AppContext {
	eol := a.Config().EOL

	return a.WriteErrString(
		fmt.Sprintf("%s%s", s, eol),
	)
}

// WriteErrString writes string data to standard error
// of this app
func (a *AppContext) WriteErrString(s string) *AppContext {
	a.WriteErr(([]byte)(s))
	return a
}

// WriteF writes formatted string data to standard output
// of this app
func (a *AppContext) WriteF(format string, args ...any) *AppContext {
	return a.WriteString(
		fmt.Sprintf(format, args...),
	)
}

// WriteLn writes string data to standard output
// of this app and adds EOL
func (a *AppContext) WriteLn(s string) *AppContext {
	eol := a.Config().EOL

	fmt.Fprintf(a, "%s%s", s, eol)
	return a
}

// WriteString writes string data to standard output
// of this app
func (a *AppContext) WriteString(s string) *AppContext {
	a.Write(([]byte)(s))
	return a
}
