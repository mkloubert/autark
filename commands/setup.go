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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mkloubert/autark/app"
	"github.com/mkloubert/autark/utils"
	"github.com/spf13/cobra"
)

const (
	registryContainerName = "autark-registry"
	registryImage         = "registry:2"
)

// SetupOptions contains options for the setup command
type SetupOptions struct {
	RegistryPort int
}

func checkDockerDaemonRunning() error {
	output, err := utils.RunCommand("docker", "info")
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "Cannot connect to the Docker daemon") ||
			strings.Contains(outputStr, "Is the docker daemon running") {
			return fmt.Errorf("Docker daemon is not running. Please start Docker first")
		}
		if outputStr != "" {
			return fmt.Errorf("Docker error: %s", outputStr)
		}
		return fmt.Errorf("Docker daemon is not accessible: %w", err)
	}
	return nil
}

func checkRegistryRunning(a *app.AppContext, port int) (bool, error) {
	if !utils.CommandExists("docker") {
		return false, fmt.Errorf("docker is not installed")
	}

	// Check if Docker daemon is running
	if err := checkDockerDaemonRunning(); err != nil {
		return false, err
	}

	// Check if container exists and is running
	output, err := utils.RunCommand("docker", "ps", "--filter", fmt.Sprintf("name=%s", registryContainerName), "--format", "{{.Status}}")
	if err != nil {
		return false, fmt.Errorf("failed to check docker containers: %w", err)
	}

	status := strings.TrimSpace(string(output))
	if status == "" {
		return false, nil
	}

	// Check if the status indicates running
	return strings.HasPrefix(strings.ToLower(status), "up"), nil
}

func initSetupCommand(a *app.AppContext) {
	rootCmd := a.RootCommand()

	opts := &SetupOptions{}

	setupCmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"s"},
		Short:   "Setup local Docker registry",
		Long:    `Sets up a local Docker registry as a background service. If not already running, it will be installed and configured to start automatically on system boot.`,
		Run: func(cmd *cobra.Command, args []string) {
			runSetup(a, opts)
		},
	}

	setupCmd.Flags().IntVarP(&opts.RegistryPort, "registry-port", "", 5000, "Port for the local Docker registry")

	rootCmd.AddCommand(setupCmd)
}

func installRegistry(a *app.AppContext, port int) error {
	a.WriteLn("Installing Docker registry...")

	// First, remove any existing container with the same name (stopped or otherwise)
	_ = exec.Command("docker", "rm", "-f", registryContainerName).Run()

	// Run the registry container with restart policy
	cmd := exec.Command("docker", "run",
		"-d",
		"--name", registryContainerName,
		"--restart=always",
		"-p", fmt.Sprintf("%d:5000", port),
		registryImage,
	)
	cmd.Stdout = a.Stdout()
	cmd.Stderr = a.Stderr()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start registry container: %w", err)
	}

	return nil
}

func runSetup(a *app.AppContext, opts *SetupOptions) {
	a.WriteLn("Checking Docker registry status...")
	a.WriteLn("")

	port := opts.RegistryPort
	a.D("Using registry port: %d", port)

	// Check if Docker is available
	if !utils.CommandExists("docker") {
		a.WriteErrLn("Docker is not installed. Please run 'autark doctor --repair' first.")
		os.Exit(1)
		return
	}

	// Check if registry is already running
	running, err := checkRegistryRunning(a, port)
	if err != nil {
		a.WriteErrLn(fmt.Sprintf("Error checking registry status: %s", err.Error()))
		os.Exit(1)
		return
	}

	if running {
		a.WriteF("Docker registry is already running on port %d.", port)
		a.WriteLn("")
		return
	}

	a.WriteF("Docker registry is not running on port %d.", port)
	a.WriteLn("")
	a.WriteLn("")

	// Install the registry
	if err := installRegistry(a, port); err != nil {
		a.WriteErrLn(fmt.Sprintf("Failed to install registry: %s", err.Error()))
		os.Exit(1)
		return
	}

	// Verify the registry is running
	running, err = checkRegistryRunning(a, port)
	if err != nil {
		a.WriteErrLn(fmt.Sprintf("Error verifying registry status: %s", err.Error()))
		os.Exit(1)
		return
	}

	if !running {
		a.WriteErrLn("Registry container started but is not running. Please check Docker logs.")
		os.Exit(1)
		return
	}

	a.WriteLn("")
	a.WriteF("Docker registry successfully installed and running on port %d.", port)
	a.WriteLn("")
	a.WriteLn("The registry will automatically restart on system boot.")
}
