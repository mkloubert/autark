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
	"runtime"
	"strings"

	"github.com/mkloubert/autark/app"
	"github.com/mkloubert/autark/utils"
	"github.com/spf13/cobra"
)

// DoctorOptions contains options for the doctor command
type DoctorOptions struct {
	Repair bool
}

// DoctorResult contains the result of a tool check
type DoctorResult struct {
	Name      string
	Installed bool
	Version   string
	Error     error
}

func checkDocker() *DoctorResult {
	result := &DoctorResult{
		Name:      "docker",
		Installed: false,
	}

	if !utils.CommandExists("docker") {
		return result
	}

	output, err := utils.RunCommand("docker", "--version")
	if err != nil {
		result.Error = err
		return result
	}

	result.Installed = true
	result.Version = strings.TrimSpace(string(output))
	return result
}

func checkDockerDaemon(dockerResult *DoctorResult) *DoctorResult {
	result := &DoctorResult{
		Name:      "docker daemon",
		Installed: false,
	}

	// If docker is not installed, daemon check is not applicable
	if !dockerResult.Installed {
		result.Error = fmt.Errorf("docker not installed")
		return result
	}

	if isDockerDaemonRunning() {
		result.Installed = true
		result.Version = "running"
	} else {
		result.Error = fmt.Errorf("not running")
	}

	return result
}

func checkGit() *DoctorResult {
	result := &DoctorResult{
		Name:      "git",
		Installed: false,
	}

	if !utils.CommandExists("git") {
		return result
	}

	output, err := utils.RunCommand("git", "--version")
	if err != nil {
		result.Error = err
		return result
	}

	result.Installed = true
	result.Version = strings.TrimSpace(string(output))
	return result
}

func checkRootPrivileges() *DoctorResult {
	result := &DoctorResult{
		Name:      "root/admin privileges",
		Installed: false,
	}

	if utils.IsRoot() {
		result.Installed = true
		if runtime.GOOS == "windows" {
			result.Version = "administrator"
		} else {
			result.Version = "root"
		}
	} else {
		if runtime.GOOS == "windows" {
			result.Error = fmt.Errorf("not running as administrator")
		} else {
			result.Error = fmt.Errorf("not running as root")
		}
	}

	return result
}

func ensureDockerDaemonRunning(a *app.AppContext) error {
	if isDockerDaemonRunning() {
		a.D("Docker daemon is already running")
		return nil
	}

	a.WriteLn("Docker daemon is not running. Attempting to start...")

	if err := startDockerDaemon(a); err != nil {
		return fmt.Errorf("failed to start docker daemon: %w", err)
	}

	// Verify daemon is now running
	if !isDockerDaemonRunning() {
		return fmt.Errorf("docker daemon failed to start")
	}

	a.WriteLn("Docker daemon started successfully.")
	return nil
}

func getVersionCodename() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			value := strings.TrimPrefix(line, "VERSION_CODENAME=")
			return strings.Trim(value, "\"'")
		}
	}

	return ""
}

func initDoctorCommand(a *app.AppContext) {
	rootCmd := a.RootCommand()

	opts := &DoctorOptions{}

	doctorCmd := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"doc", "d"},
		Short:   "Check system requirements",
		Long:    `Checks if all required tools (git, docker) are installed and optionally repairs missing dependencies.`,
		Run: func(cmd *cobra.Command, args []string) {
			runDoctor(a, opts)
		},
	}

	doctorCmd.Flags().BoolVarP(&opts.Repair, "repair", "r", false, "Install missing dependencies")

	rootCmd.AddCommand(doctorCmd)
}

func installDockerAlpine(a *app.AppContext) error {
	a.D("Installing Docker on Alpine Linux...")

	commands := [][]string{
		{"apk", "add", "docker", "docker-cli", "containerd"},
		{"rc-update", "add", "docker", "boot"},
		{"service", "docker", "start"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerArch(a *app.AppContext) error {
	a.D("Installing Docker on Arch Linux...")

	commands := [][]string{
		{"pacman", "-Sy", "--noconfirm", "docker", "docker-compose"},
		{"systemctl", "enable", "--now", "docker"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerByPackageManager(a *app.AppContext) error {
	a.D("Installing Docker via package manager fallback...")

	switch a.Platform().PackageManager {
	case utils.PkgMgrSnap:
		return runInstallCommandDirect("snap", "install", "docker")
	case utils.PkgMgrFlatpak:
		return fmt.Errorf("docker cannot be installed via flatpak, please install docker manually")
	default:
		return fmt.Errorf("docker installation not supported for package manager: %s", a.Platform().PackageManager)
	}
}

func installDockerDebian(a *app.AppContext) error {
	a.D("Installing Docker on Debian/Ubuntu...")

	// Determine the correct distro name for Docker repo
	distroName := "debian"
	if a.Platform().LinuxDistro == utils.DistroUbuntu {
		distroName = "ubuntu"
	}

	commands := [][]string{
		{"apt-get", "update", "-qq"},
		{"apt-get", "install", "-y", "-qq", "ca-certificates", "curl", "gnupg"},
		{"install", "-m", "0755", "-d", "/etc/apt/keyrings"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	// Download GPG key
	gpgURL := fmt.Sprintf("https://download.docker.com/linux/%s/gpg", distroName)
	curlCmd := exec.Command("curl", "-fsSL", gpgURL, "-o", "/etc/apt/keyrings/docker.asc")
	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("failed to download docker GPG key: %w", err)
	}

	// Get version codename
	versionCodename := getVersionCodename()
	if versionCodename == "" {
		return fmt.Errorf("could not determine version codename")
	}

	// Get architecture
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	// Add Docker repository
	repoLine := fmt.Sprintf("deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/%s %s stable",
		arch, distroName, versionCodename)

	if err := os.WriteFile("/etc/apt/sources.list.d/docker.list", []byte(repoLine+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write docker.list: %w", err)
	}

	// Update and install Docker
	finalCommands := [][]string{
		{"apt-get", "update", "-qq"},
		{"apt-get", "install", "-y", "-qq", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
	}

	for _, cmd := range finalCommands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerFedora(a *app.AppContext) error {
	a.D("Installing Docker on Fedora/RHEL...")

	commands := [][]string{
		{"dnf", "config-manager", "addrepo", "--from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo"},
		{"dnf", "install", "-y", "-q", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
		{"systemctl", "enable", "--now", "docker"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerGentoo(a *app.AppContext) error {
	a.D("Installing Docker on Gentoo...")

	commands := [][]string{
		{"emerge", "--quiet", "app-containers/docker", "app-containers/docker-compose"},
		{"rc-update", "add", "docker", "default"},
		{"service", "docker", "start"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerOpenSUSE(a *app.AppContext) error {
	a.D("Installing Docker on openSUSE...")

	commands := [][]string{
		{"zypper", "install", "-y", "docker", "docker-compose"},
		{"systemctl", "enable", "--now", "docker"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func installDockerVoid(a *app.AppContext) error {
	a.D("Installing Docker on Void Linux...")

	commands := [][]string{
		{"xbps-install", "-y", "docker", "docker-compose"},
		{"ln", "-s", "/etc/sv/docker", "/var/service/"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	return nil
}

func isDockerDaemonRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func printResult(a *app.AppContext, r *DoctorResult) {
	if r.Installed {
		version := r.Version
		if version == "" {
			version = "installed"
		}
		a.WriteF("[OK] %s: %s", r.Name, version)
	} else {
		msg := "not found"
		if r.Error != nil {
			msg = r.Error.Error()
		}
		a.WriteErrF("[ERROR] %s: %s", r.Name, msg)
	}
	a.WriteLn("")
}

func repairDocker(a *app.AppContext) error {
	a.WriteLn("Installing docker...")

	switch a.Platform().OS {
	case utils.OSLinux:
		return repairDockerLinux(a)
	case utils.OSDarwin:
		return repairDockerDarwin(a)
	case utils.OSWindows:
		return repairDockerWindows(a)
	case utils.OSFreeBSD:
		return repairDockerBSD(a)
	default:
		return fmt.Errorf("docker installation not supported on %s", a.Platform().OS)
	}
}

func repairDockerBSD(a *app.AppContext) error {
	a.D("Installing Docker on BSD...")

	if a.Platform().PackageManager != utils.PkgMgrPkg {
		return fmt.Errorf("pkg is required to install Docker on BSD")
	}

	// Note: Docker has limited support on BSD, this installs the available packages
	commands := [][]string{
		{"pkg", "install", "-y", "docker"},
	}

	for _, cmd := range commands {
		if err := runInstallCommandDirect(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to run %s: %w", cmd[0], err)
		}
	}

	a.WriteLn("Note: Docker support on BSD is limited. Consider using jails or bhyve for containerization.")
	return nil
}

func repairDockerDarwin(a *app.AppContext) error {
	a.D("Installing Docker on macOS...")

	switch a.Platform().PackageManager {
	case utils.PkgMgrBrew:
		// Install Docker Desktop via brew cask
		if err := runInstallCommandDirect("brew", "install", "--cask", "docker"); err != nil {
			return fmt.Errorf("failed to install Docker Desktop: %w", err)
		}
		a.WriteLn("Docker Desktop installed. Please open Docker Desktop from Applications to complete setup.")
		return nil
	case utils.PkgMgrPort:
		// MacPorts has docker available
		if err := runInstallCommandDirect("port", "install", "docker"); err != nil {
			return fmt.Errorf("failed to install docker via MacPorts: %w", err)
		}
		a.WriteLn("Docker installed via MacPorts. You may need to configure it manually.")
		return nil
	default:
		return fmt.Errorf("homebrew or MacPorts is required to install Docker on macOS")
	}
}

func repairDockerLinux(a *app.AppContext) error {
	switch a.Platform().LinuxDistro {
	case utils.DistroDebian, utils.DistroUbuntu:
		return installDockerDebian(a)
	case utils.DistroFedora, utils.DistroRHEL, utils.DistroCentOS:
		return installDockerFedora(a)
	case utils.DistroArch:
		return installDockerArch(a)
	case utils.DistroAlpine:
		return installDockerAlpine(a)
	case utils.DistroOpenSUSE:
		return installDockerOpenSUSE(a)
	case utils.DistroGentoo:
		return installDockerGentoo(a)
	case utils.DistroVoid:
		return installDockerVoid(a)
	default:
		// Try fallback based on package manager
		return installDockerByPackageManager(a)
	}
}

func repairDockerWindows(a *app.AppContext) error {
	a.D("Installing Docker on Windows...")

	switch a.Platform().PackageManager {
	case utils.PkgMgrWinget:
		return runInstallCommandDirect("winget", "install", "--id", "Docker.DockerDesktop", "-e", "--silent")
	case utils.PkgMgrChoco:
		return runInstallCommandDirect("choco", "install", "docker-desktop", "-y")
	default:
		return fmt.Errorf("winget or chocolatey is required to install Docker on Windows")
	}
}

func repairGit(a *app.AppContext) error {
	a.WriteLn("Installing git...")

	switch a.Platform().PackageManager {
	case utils.PkgMgrApt:
		return runInstallCommand("apt-get", "update", "-qq", "&&", "apt-get", "install", "-y", "-qq", "git")
	case utils.PkgMgrDnf:
		return runInstallCommand("dnf", "install", "-y", "-q", "git")
	case utils.PkgMgrPacman:
		return runInstallCommand("pacman", "-Sy", "--noconfirm", "git")
	case utils.PkgMgrApk:
		return runInstallCommand("apk", "add", "--quiet", "git")
	case utils.PkgMgrZypper:
		return runInstallCommand("zypper", "install", "-y", "-q", "git")
	case utils.PkgMgrEmerge:
		return runInstallCommandDirect("emerge", "--quiet", "dev-vcs/git")
	case utils.PkgMgrXbpsInstall:
		return runInstallCommandDirect("xbps-install", "-y", "git")
	case utils.PkgMgrSnap:
		return runInstallCommandDirect("snap", "install", "git")
	case utils.PkgMgrFlatpak:
		return fmt.Errorf("git cannot be installed via flatpak, please install git manually")
	case utils.PkgMgrBrew:
		return runInstallCommandDirect("brew", "install", "git")
	case utils.PkgMgrPort:
		return runInstallCommandDirect("port", "install", "git")
	case utils.PkgMgrPkg:
		return runInstallCommandDirect("pkg", "install", "-y", "git")
	case utils.PkgMgrWinget:
		return runInstallCommandDirect("winget", "install", "--id", "Git.Git", "-e", "--silent")
	case utils.PkgMgrChoco:
		return runInstallCommandDirect("choco", "install", "git", "-y")
	default:
		return fmt.Errorf("unsupported package manager: %s", a.Platform().PackageManager)
	}
}

func runDoctor(a *app.AppContext, opts *DoctorOptions) {
	a.WriteLn("Checking system requirements...")
	a.WriteLn("")

	platform := a.Platform()

	a.D("Detected OS: %s", platform.OS)
	a.D("Detected Arch: %s", platform.Arch)
	if platform.OS == utils.OSLinux {
		a.D("Detected Linux Distro: %s (%s)", platform.LinuxDistro, platform.LinuxDistroID)
	}
	a.D("Detected Package Manager: %s", platform.PackageManager)
	a.D("")

	results := make([]*DoctorResult, 0)

	// Check root/admin privileges
	rootResult := checkRootPrivileges()
	results = append(results, rootResult)
	printResult(a, rootResult)

	// Check git
	gitResult := checkGit()
	results = append(results, gitResult)
	printResult(a, gitResult)

	// Check docker
	dockerResult := checkDocker()
	results = append(results, dockerResult)
	printResult(a, dockerResult)

	// Check docker daemon status
	dockerDaemonResult := checkDockerDaemon(dockerResult)
	results = append(results, dockerDaemonResult)
	printResult(a, dockerDaemonResult)

	a.WriteLn("")

	// Count issues
	issues := 0
	for _, r := range results {
		if !r.Installed {
			issues++
		}
	}

	if issues == 0 {
		a.WriteLn("All requirements satisfied!")
		return
	}

	a.WriteF("Found %d issue(s).", issues)
	a.WriteLn("")

	if !opts.Repair {
		a.WriteLn("")
		a.WriteLn("Run 'autark doctor --repair' to fix missing dependencies.")
		os.Exit(1)
		return
	}

	// Check for root/admin privileges before attempting repair
	if !utils.IsRoot() {
		a.WriteLn("")
		if runtime.GOOS == "windows" {
			a.WriteErrLn("Error: --repair requires administrator privileges.")
			a.WriteErrLn("Please run this command as Administrator.")
		} else {
			a.WriteErrLn("Error: --repair requires root privileges.")
			a.WriteErrLn("Please run this command with sudo.")
		}
		os.Exit(1)
		return
	}

	a.WriteLn("")
	a.WriteLn("Attempting to repair...")
	a.WriteLn("")

	repairErrors := 0

	// Repair git if needed
	if !gitResult.Installed {
		if err := repairGit(a); err != nil {
			a.WriteErrLn(fmt.Sprintf("Failed to install git: %s", err.Error()))
			repairErrors++
		} else {
			a.WriteLn("git installed successfully.")
		}
	}

	// Repair docker if needed
	if !dockerResult.Installed {
		if err := repairDocker(a); err != nil {
			a.WriteErrLn(fmt.Sprintf("Failed to install docker: %s", err.Error()))
			repairErrors++
		} else {
			a.WriteLn("docker installed successfully.")
		}
	}

	// Start docker daemon if needed
	if !dockerDaemonResult.Installed {
		if err := ensureDockerDaemonRunning(a); err != nil {
			a.WriteErrLn(fmt.Sprintf("Failed to start docker daemon: %s", err.Error()))
			repairErrors++
		}
	}

	if repairErrors > 0 {
		a.WriteLn("")
		a.WriteErrF("Repair completed with %d error(s).", repairErrors)
		a.WriteLn("")
		os.Exit(1)
	}

	a.WriteLn("")
	a.WriteLn("Repair completed successfully.")
}

func runInstallCommand(name string, args ...string) error {
	// Handle commands with shell operators
	cmdStr := name + " " + strings.Join(args, " ")
	if strings.Contains(cmdStr, "&&") || strings.Contains(cmdStr, "|") {
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return runInstallCommandDirect(name, args...)
}

func runInstallCommandDirect(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startDockerDaemon(a *app.AppContext) error {
	switch a.Platform().OS {
	case utils.OSLinux:
		return startDockerDaemonLinux(a)
	case utils.OSDarwin:
		return startDockerDaemonDarwin(a)
	case utils.OSWindows:
		return startDockerDaemonWindows(a)
	default:
		return fmt.Errorf("starting docker daemon not supported on %s", a.Platform().OS)
	}
}

func startDockerDaemonLinux(a *app.AppContext) error {
	// Try systemd first (most common)
	if utils.CommandExists("systemctl") {
		a.D("Attempting to start docker via systemctl...")
		if err := runInstallCommandDirect("systemctl", "start", "docker"); err == nil {
			return nil
		}
	}

	// Try OpenRC (Alpine, Gentoo)
	if utils.CommandExists("rc-service") {
		a.D("Attempting to start docker via rc-service...")
		if err := runInstallCommandDirect("rc-service", "docker", "start"); err == nil {
			return nil
		}
	}

	// Try service command (generic fallback)
	if utils.CommandExists("service") {
		a.D("Attempting to start docker via service...")
		if err := runInstallCommandDirect("service", "docker", "start"); err == nil {
			return nil
		}
	}

	// Try starting dockerd directly as last resort
	a.D("Attempting to start dockerd directly...")
	cmd := exec.Command("dockerd")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start docker daemon: %w", err)
	}

	// Wait a moment for daemon to initialize
	return nil
}

func startDockerDaemonDarwin(a *app.AppContext) error {
	a.D("Attempting to start Docker Desktop on macOS...")

	// Try to open Docker Desktop
	if err := runInstallCommandDirect("open", "-a", "Docker"); err != nil {
		return fmt.Errorf("failed to start Docker Desktop: %w", err)
	}

	a.WriteLn("Docker Desktop is starting. Please wait for it to initialize...")
	return nil
}

func startDockerDaemonWindows(a *app.AppContext) error {
	a.D("Attempting to start Docker Desktop on Windows...")

	// Try to start Docker Desktop via PowerShell
	cmd := exec.Command("powershell", "-Command", "Start-Process 'C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe'")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Docker Desktop: %w", err)
	}

	a.WriteLn("Docker Desktop is starting. Please wait for it to initialize...")
	return nil
}
