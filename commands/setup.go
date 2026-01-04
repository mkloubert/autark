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
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
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
	NoFirewall   bool
	NoSSH        bool
}

// FirewallInfo contains information about the detected firewall
type FirewallInfo struct {
	Name      string
	Installed bool
	Command   string
}

// SSHInfo contains information about the detected SSH server
type SSHInfo struct {
	Name      string
	Installed bool
	Running   bool
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

func checkRegistryRunning() (bool, error) {
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

func checkFirewall() *FirewallInfo {
	switch runtime.GOOS {
	case "linux":
		return checkFirewallLinux()
	case "darwin":
		return checkFirewallDarwin()
	case "windows":
		return checkFirewallWindows()
	default:
		return &FirewallInfo{Name: "unknown", Installed: false}
	}
}

func checkFirewallLinux() *FirewallInfo {
	// Check for ufw (Ubuntu/Debian)
	if utils.CommandExists("ufw") {
		return &FirewallInfo{Name: "ufw", Installed: true, Command: "ufw"}
	}

	// Check for firewalld (Fedora/RHEL/CentOS)
	if utils.CommandExists("firewall-cmd") {
		return &FirewallInfo{Name: "firewalld", Installed: true, Command: "firewall-cmd"}
	}

	// Check for iptables (generic fallback, usually always present)
	if utils.CommandExists("iptables") {
		return &FirewallInfo{Name: "iptables", Installed: true, Command: "iptables"}
	}

	// Check for nftables (modern replacement for iptables)
	if utils.CommandExists("nft") {
		return &FirewallInfo{Name: "nftables", Installed: true, Command: "nft"}
	}

	return &FirewallInfo{Name: "ufw", Installed: false, Command: "ufw"}
}

func checkFirewallDarwin() *FirewallInfo {
	// macOS has pf (Packet Filter) built-in
	if utils.CommandExists("pfctl") {
		return &FirewallInfo{Name: "pf", Installed: true, Command: "pfctl"}
	}

	return &FirewallInfo{Name: "pf", Installed: true, Command: "pfctl"}
}

func checkFirewallWindows() *FirewallInfo {
	// Windows Firewall is always available via netsh
	cmd := exec.Command("netsh", "advfirewall", "show", "allprofiles", "state")
	if err := cmd.Run(); err == nil {
		return &FirewallInfo{Name: "Windows Firewall", Installed: true, Command: "netsh"}
	}

	return &FirewallInfo{Name: "Windows Firewall", Installed: true, Command: "netsh"}
}

func checkSSH() *SSHInfo {
	switch runtime.GOOS {
	case "linux":
		return checkSSHLinux()
	case "darwin":
		return checkSSHDarwin()
	case "windows":
		return checkSSHWindows()
	default:
		return &SSHInfo{Name: "unknown", Installed: false, Running: false}
	}
}

func checkSSHLinux() *SSHInfo {
	info := &SSHInfo{Name: "openssh", Installed: false, Running: false}

	// Check if sshd is installed
	if utils.CommandExists("sshd") {
		info.Installed = true
	} else if _, err := os.Stat("/usr/sbin/sshd"); err == nil {
		info.Installed = true
	}

	// Check if sshd is running
	if info.Installed {
		// Try systemctl first
		if utils.CommandExists("systemctl") {
			cmd := exec.Command("systemctl", "is-active", "--quiet", "sshd")
			if cmd.Run() == nil {
				info.Running = true
			} else {
				// Try ssh service name (used on Debian/Ubuntu)
				cmd = exec.Command("systemctl", "is-active", "--quiet", "ssh")
				if cmd.Run() == nil {
					info.Running = true
				}
			}
		}

		// Try rc-service (Alpine/OpenRC)
		if !info.Running && utils.CommandExists("rc-service") {
			cmd := exec.Command("rc-service", "sshd", "status")
			if cmd.Run() == nil {
				info.Running = true
			}
		}

		// Check if process is running
		if !info.Running {
			cmd := exec.Command("pgrep", "-x", "sshd")
			if cmd.Run() == nil {
				info.Running = true
			}
		}
	}

	return info
}

func checkSSHDarwin() *SSHInfo {
	info := &SSHInfo{Name: "openssh", Installed: true, Running: false}

	// macOS has SSH built-in, check if Remote Login is enabled
	cmd := exec.Command("systemsetup", "-getremotelogin")
	output, err := cmd.Output()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "on") {
		info.Running = true
	}

	return info
}

func checkSSHWindows() *SSHInfo {
	info := &SSHInfo{Name: "openssh", Installed: false, Running: false}

	// Check if OpenSSH Server is installed on Windows
	cmd := exec.Command("powershell", "-Command",
		"Get-WindowsCapability -Online | Where-Object Name -like 'OpenSSH.Server*' | Select-Object -ExpandProperty State")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "Installed" {
		info.Installed = true
	}

	// Check if sshd service is running
	if info.Installed {
		cmd = exec.Command("powershell", "-Command",
			"(Get-Service sshd -ErrorAction SilentlyContinue).Status")
		output, err = cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "Running" {
			info.Running = true
		}
	}

	return info
}

func configureSSHPort(port int) error {
	if port == 22 {
		return nil // Default port, no configuration needed
	}

	// Read current sshd_config
	configPath := "/etc/ssh/sshd_config"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read sshd_config: %w", err)
	}

	// Check if Port line exists and modify it
	lines := strings.Split(string(content), "\n")
	portConfigured := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Port ") || strings.HasPrefix(trimmed, "#Port ") {
			lines[i] = fmt.Sprintf("Port %d", port)
			portConfigured = true
			break
		}
	}

	if !portConfigured {
		// Add Port configuration at the beginning
		lines = append([]string{fmt.Sprintf("Port %d", port)}, lines...)
	}

	// Write back
	err = os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to write sshd_config: %w", err)
	}

	return nil
}

// generateRandomPort generates a random available port > 1024
func generateRandomPort() int {
	const minPort = 1025
	const maxPort = 65535
	const maxAttempts = 100

	for i := 0; i < maxAttempts; i++ {
		port := minPort + rand.Intn(maxPort-minPort)
		if isTCPPortAvailable(port) {
			return port
		}
	}

	// Fallback to a commonly used alternative SSH port
	return 2222
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
	setupCmd.Flags().BoolVarP(&opts.NoFirewall, "no-firewall", "", false, "Skip firewall check and installation")
	setupCmd.Flags().BoolVarP(&opts.NoSSH, "no-ssh", "", false, "Skip SSH server check and installation")

	rootCmd.AddCommand(setupCmd)
}

func installFirewall(a *app.AppContext) error {
	platform := a.Platform()

	switch platform.OS {
	case utils.OSLinux:
		return installFirewallLinux(a)
	case utils.OSDarwin:
		a.WriteLn("macOS has pf (Packet Filter) built-in. No installation required.")
		return nil
	case utils.OSWindows:
		a.WriteLn("Windows Firewall is built-in. No installation required.")
		return nil
	default:
		return fmt.Errorf("firewall installation not supported on %s", platform.OS)
	}
}

func installFirewallArch(a *app.AppContext) error {
	a.D("Installing ufw on Arch Linux...")

	if err := runInstallCommandDirect("pacman", "-Sy", "--noconfirm", "ufw"); err != nil {
		return fmt.Errorf("failed to install ufw: %w", err)
	}

	return nil
}

func installFirewallAlpine(a *app.AppContext) error {
	a.D("Installing iptables on Alpine Linux...")

	if err := runInstallCommandDirect("apk", "add", "iptables"); err != nil {
		return fmt.Errorf("failed to install iptables: %w", err)
	}

	return nil
}

func installFirewallByPackageManager(a *app.AppContext) error {
	platform := a.Platform()

	switch platform.PackageManager {
	case utils.PkgMgrApt:
		return runInstallCommandDirect("apt-get", "install", "-y", "-qq", "ufw")
	case utils.PkgMgrDnf:
		return runInstallCommandDirect("dnf", "install", "-y", "-q", "firewalld")
	case utils.PkgMgrPacman:
		return runInstallCommandDirect("pacman", "-Sy", "--noconfirm", "ufw")
	case utils.PkgMgrApk:
		return runInstallCommandDirect("apk", "add", "iptables")
	case utils.PkgMgrZypper:
		return runInstallCommandDirect("zypper", "install", "-y", "firewalld")
	default:
		return fmt.Errorf("firewall installation not supported for package manager: %s", platform.PackageManager)
	}
}

func installFirewallDebian(a *app.AppContext) error {
	a.D("Installing ufw on Debian/Ubuntu...")

	if err := runInstallCommandDirect("apt-get", "update", "-qq"); err != nil {
		return fmt.Errorf("failed to update package list: %w", err)
	}

	if err := runInstallCommandDirect("apt-get", "install", "-y", "-qq", "ufw"); err != nil {
		return fmt.Errorf("failed to install ufw: %w", err)
	}

	return nil
}

func installFirewallFedora(a *app.AppContext) error {
	a.D("Installing firewalld on Fedora/RHEL...")

	if err := runInstallCommandDirect("dnf", "install", "-y", "-q", "firewalld"); err != nil {
		return fmt.Errorf("failed to install firewalld: %w", err)
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "firewalld"); err != nil {
		return fmt.Errorf("failed to enable firewalld: %w", err)
	}

	return nil
}

func installFirewallLinux(a *app.AppContext) error {
	platform := a.Platform()

	a.WriteLn("Installing firewall...")

	switch platform.LinuxDistro {
	case utils.DistroDebian, utils.DistroUbuntu:
		return installFirewallDebian(a)
	case utils.DistroFedora, utils.DistroRHEL, utils.DistroCentOS:
		return installFirewallFedora(a)
	case utils.DistroArch:
		return installFirewallArch(a)
	case utils.DistroAlpine:
		return installFirewallAlpine(a)
	case utils.DistroOpenSUSE:
		return installFirewallOpenSUSE(a)
	case utils.DistroGentoo:
		return installFirewallGentoo(a)
	case utils.DistroVoid:
		return installFirewallVoid(a)
	default:
		return installFirewallByPackageManager(a)
	}
}

func installFirewallGentoo(a *app.AppContext) error {
	a.D("Installing iptables on Gentoo...")

	if err := runInstallCommandDirect("emerge", "--quiet", "net-firewall/iptables"); err != nil {
		return fmt.Errorf("failed to install iptables: %w", err)
	}

	return nil
}

func installFirewallOpenSUSE(a *app.AppContext) error {
	a.D("Installing firewalld on openSUSE...")

	if err := runInstallCommandDirect("zypper", "install", "-y", "firewalld"); err != nil {
		return fmt.Errorf("failed to install firewalld: %w", err)
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "firewalld"); err != nil {
		return fmt.Errorf("failed to enable firewalld: %w", err)
	}

	return nil
}

func installFirewallVoid(a *app.AppContext) error {
	a.D("Installing iptables on Void Linux...")

	if err := runInstallCommandDirect("xbps-install", "-y", "iptables"); err != nil {
		return fmt.Errorf("failed to install iptables: %w", err)
	}

	return nil
}

func installSSH(a *app.AppContext, port int) error {
	platform := a.Platform()

	switch platform.OS {
	case utils.OSLinux:
		return installSSHLinux(a, port)
	case utils.OSDarwin:
		return installSSHDarwin(a, port)
	case utils.OSWindows:
		return installSSHWindows(a, port)
	default:
		return fmt.Errorf("SSH installation not supported on %s", platform.OS)
	}
}

func installSSHLinux(a *app.AppContext, port int) error {
	platform := a.Platform()

	a.WriteLn("Installing OpenSSH server...")

	switch platform.LinuxDistro {
	case utils.DistroDebian, utils.DistroUbuntu:
		return installSSHDebian(a, port)
	case utils.DistroFedora, utils.DistroRHEL, utils.DistroCentOS:
		return installSSHFedora(a, port)
	case utils.DistroArch:
		return installSSHArch(a, port)
	case utils.DistroAlpine:
		return installSSHAlpine(a, port)
	case utils.DistroOpenSUSE:
		return installSSHOpenSUSE(a, port)
	case utils.DistroGentoo:
		return installSSHGentoo(a, port)
	case utils.DistroVoid:
		return installSSHVoid(a, port)
	default:
		return installSSHByPackageManager(a, port)
	}
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

func installSSHAlpine(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Alpine Linux...")

	if err := runInstallCommandDirect("apk", "add", "openssh"); err != nil {
		return fmt.Errorf("failed to install openssh: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("rc-update", "add", "sshd"); err != nil {
		return fmt.Errorf("failed to enable sshd service: %w", err)
	}

	if err := runInstallCommandDirect("service", "sshd", "start"); err != nil {
		return fmt.Errorf("failed to start sshd service: %w", err)
	}

	return nil
}

func installSSHArch(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Arch Linux...")

	if err := runInstallCommandDirect("pacman", "-Sy", "--noconfirm", "openssh"); err != nil {
		return fmt.Errorf("failed to install openssh: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "sshd"); err != nil {
		return fmt.Errorf("failed to enable sshd service: %w", err)
	}

	return nil
}

func installSSHByPackageManager(a *app.AppContext, port int) error {
	platform := a.Platform()

	switch platform.PackageManager {
	case utils.PkgMgrApt:
		if err := runInstallCommandDirect("apt-get", "install", "-y", "-qq", "openssh-server"); err != nil {
			return err
		}
		if err := configureSSHPort(port); err != nil {
			a.W("Failed to configure SSH port: %s", err.Error())
		}
		return runInstallCommandDirect("systemctl", "enable", "--now", "ssh")
	case utils.PkgMgrDnf:
		if err := runInstallCommandDirect("dnf", "install", "-y", "-q", "openssh-server"); err != nil {
			return err
		}
		if err := configureSSHPort(port); err != nil {
			a.W("Failed to configure SSH port: %s", err.Error())
		}
		return runInstallCommandDirect("systemctl", "enable", "--now", "sshd")
	case utils.PkgMgrPacman:
		if err := runInstallCommandDirect("pacman", "-Sy", "--noconfirm", "openssh"); err != nil {
			return err
		}
		if err := configureSSHPort(port); err != nil {
			a.W("Failed to configure SSH port: %s", err.Error())
		}
		return runInstallCommandDirect("systemctl", "enable", "--now", "sshd")
	case utils.PkgMgrApk:
		if err := runInstallCommandDirect("apk", "add", "openssh"); err != nil {
			return err
		}
		if err := configureSSHPort(port); err != nil {
			a.W("Failed to configure SSH port: %s", err.Error())
		}
		return runInstallCommandDirect("rc-update", "add", "sshd")
	default:
		return fmt.Errorf("SSH installation not supported for package manager: %s", platform.PackageManager)
	}
}

func installSSHDarwin(a *app.AppContext, port int) error {
	a.WriteLn("Enabling Remote Login (SSH) on macOS...")

	// Enable Remote Login via systemsetup (requires admin privileges)
	if err := runInstallCommandDirect("systemsetup", "-setremotelogin", "on"); err != nil {
		return fmt.Errorf("failed to enable Remote Login: %w", err)
	}

	if port != 22 {
		a.W("Custom SSH port configuration on macOS requires manual editing of /etc/ssh/sshd_config")
	}

	return nil
}

func installSSHDebian(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Debian/Ubuntu...")

	if err := runInstallCommandDirect("apt-get", "update", "-qq"); err != nil {
		return fmt.Errorf("failed to update package list: %w", err)
	}

	if err := runInstallCommandDirect("apt-get", "install", "-y", "-qq", "openssh-server"); err != nil {
		return fmt.Errorf("failed to install openssh-server: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "ssh"); err != nil {
		return fmt.Errorf("failed to enable ssh service: %w", err)
	}

	return nil
}

func installSSHFedora(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Fedora/RHEL...")

	if err := runInstallCommandDirect("dnf", "install", "-y", "-q", "openssh-server"); err != nil {
		return fmt.Errorf("failed to install openssh-server: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "sshd"); err != nil {
		return fmt.Errorf("failed to enable sshd service: %w", err)
	}

	return nil
}

func installSSHGentoo(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Gentoo...")

	if err := runInstallCommandDirect("emerge", "--quiet", "net-misc/openssh"); err != nil {
		return fmt.Errorf("failed to install openssh: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("rc-update", "add", "sshd", "default"); err != nil {
		return fmt.Errorf("failed to enable sshd service: %w", err)
	}

	if err := runInstallCommandDirect("service", "sshd", "start"); err != nil {
		return fmt.Errorf("failed to start sshd service: %w", err)
	}

	return nil
}

func installSSHOpenSUSE(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on openSUSE...")

	if err := runInstallCommandDirect("zypper", "install", "-y", "openssh"); err != nil {
		return fmt.Errorf("failed to install openssh: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("systemctl", "enable", "--now", "sshd"); err != nil {
		return fmt.Errorf("failed to enable sshd service: %w", err)
	}

	return nil
}

func installSSHVoid(a *app.AppContext, port int) error {
	a.D("Installing OpenSSH server on Void Linux...")

	if err := runInstallCommandDirect("xbps-install", "-y", "openssh"); err != nil {
		return fmt.Errorf("failed to install openssh: %w", err)
	}

	if err := configureSSHPort(port); err != nil {
		a.W("Failed to configure SSH port: %s", err.Error())
	}

	if err := runInstallCommandDirect("ln", "-s", "/etc/sv/sshd", "/var/service/"); err != nil {
		// Link might already exist, just warn
		a.W("Failed to enable sshd service: %s", err.Error())
	}

	return nil
}

func installSSHWindows(a *app.AppContext, port int) error {
	a.WriteLn("Installing OpenSSH server on Windows...")

	// Install OpenSSH Server capability
	cmd := exec.Command("powershell", "-Command",
		"Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install OpenSSH Server: %w", err)
	}

	// Configure port if not default
	if port != 22 {
		configCmd := exec.Command("powershell", "-Command",
			fmt.Sprintf(`$config = Get-Content $env:ProgramData\ssh\sshd_config; `+
				`$config = $config -replace '^#?Port \d+', 'Port %d'; `+
				`Set-Content $env:ProgramData\ssh\sshd_config $config`, port))
		if err := configCmd.Run(); err != nil {
			a.W("Failed to configure SSH port: %s", err.Error())
		}
	}

	// Start and enable sshd service
	startCmd := exec.Command("powershell", "-Command",
		"Start-Service sshd; Set-Service -Name sshd -StartupType Automatic")
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start sshd service: %w", err)
	}

	// Configure firewall rule
	fwCmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' "+
			"-Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort %d", port))
	if err := fwCmd.Run(); err != nil {
		a.W("Failed to configure firewall rule: %s", err.Error())
	}

	return nil
}

// isPortAvailable checks if a TCP port is available (not in use)
func isTCPPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func runSetup(a *app.AppContext, opts *SetupOptions) {
	// Check firewall status unless --no-firewall is set
	if !opts.NoFirewall {
		a.WriteLn("Checking firewall status...")

		firewallInfo := checkFirewall()

		if firewallInfo.Installed {
			a.WriteF("[OK] Firewall detected: %s", firewallInfo.Name)
			a.WriteLn("")
		} else {
			a.WriteF("[WARN] No firewall detected.")
			a.WriteLn("")
			a.WriteLn("")

			if a.PromptYesNo("Would you like to install a firewall?", true) {
				// Check for root privileges
				if !utils.IsRoot() {
					a.WriteLn("")
					if runtime.GOOS == "windows" {
						a.WriteErrLn("Error: Firewall installation requires administrator privileges.")
						a.WriteErrLn("Please run this command as Administrator.")
					} else {
						a.WriteErrLn("Error: Firewall installation requires root privileges.")
						a.WriteErrLn("Please run this command with sudo.")
					}
					os.Exit(1)
					return
				}

				if err := installFirewall(a); err != nil {
					a.WriteErrLn(fmt.Sprintf("Failed to install firewall: %s", err.Error()))
					os.Exit(1)
					return
				}

				a.WriteLn("Firewall installed successfully.")
			} else {
				a.WriteLn("Skipping firewall installation.")
			}
		}

		a.WriteLn("")
	}

	// Check SSH server status unless --no-ssh is set
	if !opts.NoSSH {
		a.WriteLn("Checking SSH server status...")

		sshInfo := checkSSH()

		if sshInfo.Installed && sshInfo.Running {
			a.WriteF("[OK] SSH server detected: %s (running)", sshInfo.Name)
			a.WriteLn("")
		} else if sshInfo.Installed {
			a.WriteF("[WARN] SSH server installed but not running: %s", sshInfo.Name)
			a.WriteLn("")
		} else {
			a.WriteF("[WARN] No SSH server detected.")
			a.WriteLn("")
			a.WriteLn("")

			if a.PromptYesNo("Would you like to install an SSH server?", true) {
				// Check for root privileges
				if !utils.IsRoot() {
					a.WriteLn("")
					if runtime.GOOS == "windows" {
						a.WriteErrLn("Error: SSH installation requires administrator privileges.")
						a.WriteErrLn("Please run this command as Administrator.")
					} else {
						a.WriteErrLn("Error: SSH installation requires root privileges.")
						a.WriteErrLn("Please run this command with sudo.")
					}
					os.Exit(1)
					return
				}

				// Generate a random available port as suggestion
				suggestedPort := generateRandomPort()
				a.WriteLn("")
				a.WriteF("Suggested SSH port: %d (random, available, > 1024)", suggestedPort)
				a.WriteLn("")

				// Ask user for the port
				sshPort := a.PromptPort("Enter SSH port", suggestedPort)

				// Verify the port is available
				if !isTCPPortAvailable(sshPort) {
					a.WriteErrLn(fmt.Sprintf("Port %d is already in use. Please choose a different port.", sshPort))
					os.Exit(1)
					return
				}

				a.WriteLn("")
				a.WriteF("Installing SSH server on port %d...", sshPort)
				a.WriteLn("")

				if err := installSSH(a, sshPort); err != nil {
					a.WriteErrLn(fmt.Sprintf("Failed to install SSH server: %s", err.Error()))
					os.Exit(1)
					return
				}

				a.WriteF("SSH server installed successfully on port %d.", sshPort)
				a.WriteLn("")
			} else {
				a.WriteLn("Skipping SSH server installation.")
			}
		}

		a.WriteLn("")
	}

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
	running, err := checkRegistryRunning()
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
	running, err = checkRegistryRunning()
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
