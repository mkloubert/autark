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
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OSType represents the operating system type
type OSType string

const (
	OSLinux   OSType = "linux"
	OSDarwin  OSType = "darwin"
	OSWindows OSType = "windows"
	OSFreeBSD OSType = "freebsd"
	OSUnknown OSType = "unknown"
)

// LinuxDistro represents the Linux distribution
type LinuxDistro string

const (
	DistroDebian   LinuxDistro = "debian"
	DistroUbuntu   LinuxDistro = "ubuntu"
	DistroFedora   LinuxDistro = "fedora"
	DistroRHEL     LinuxDistro = "rhel"
	DistroCentOS   LinuxDistro = "centos"
	DistroArch     LinuxDistro = "arch"
	DistroAlpine   LinuxDistro = "alpine"
	DistroOpenSUSE LinuxDistro = "opensuse"
	DistroGentoo   LinuxDistro = "gentoo"
	DistroVoid     LinuxDistro = "void"
	DistroUnknown  LinuxDistro = "unknown"
)

// PackageManager represents the package manager type
type PackageManager string

const (
	PkgMgrApt         PackageManager = "apt"
	PkgMgrDnf         PackageManager = "dnf"
	PkgMgrPacman      PackageManager = "pacman"
	PkgMgrApk         PackageManager = "apk"
	PkgMgrZypper      PackageManager = "zypper"
	PkgMgrEmerge      PackageManager = "emerge"
	PkgMgrXbpsInstall PackageManager = "xbps-install"
	PkgMgrSnap        PackageManager = "snap"
	PkgMgrFlatpak     PackageManager = "flatpak"
	PkgMgrBrew        PackageManager = "brew"
	PkgMgrPort        PackageManager = "port"
	PkgMgrPkg         PackageManager = "pkg"
	PkgMgrChoco       PackageManager = "choco"
	PkgMgrWinget      PackageManager = "winget"
	PkgMgrUnknown     PackageManager = "unknown"
)

// PlatformInfo contains information about the current platform
type PlatformInfo struct {
	OS             OSType
	Arch           string
	LinuxDistro    LinuxDistro
	LinuxDistroID  string
	PackageManager PackageManager
}

func (p *PlatformInfo) detectBSDPackageManager() {
	if CommandExists("pkg") {
		p.PackageManager = PkgMgrPkg
	}
}

func (p *PlatformInfo) detectDarwinPackageManager() {
	if CommandExists("brew") {
		p.PackageManager = PkgMgrBrew
	} else if CommandExists("port") {
		p.PackageManager = PkgMgrPort
	}
}

func (p *PlatformInfo) detectLinuxDistro() {
	osRelease, err := parseOSRelease("/etc/os-release")
	if err != nil {
		return
	}

	p.LinuxDistroID = osRelease["ID"]
	idLike := osRelease["ID_LIKE"]

	switch p.LinuxDistroID {
	case "debian":
		p.LinuxDistro = DistroDebian
	case "ubuntu", "linuxmint", "pop", "elementary", "zorin", "kali", "raspbian", "neon":
		p.LinuxDistro = DistroUbuntu
	case "fedora":
		p.LinuxDistro = DistroFedora
	case "rhel", "rocky", "almalinux", "ol", "amzn":
		p.LinuxDistro = DistroRHEL
	case "centos":
		p.LinuxDistro = DistroCentOS
	case "arch", "manjaro", "endeavouros", "garuda", "artix":
		p.LinuxDistro = DistroArch
	case "alpine":
		p.LinuxDistro = DistroAlpine
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		p.LinuxDistro = DistroOpenSUSE
	case "gentoo":
		p.LinuxDistro = DistroGentoo
	case "void":
		p.LinuxDistro = DistroVoid
	default:
		if strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
			p.LinuxDistro = DistroDebian
		} else if strings.Contains(idLike, "fedora") || strings.Contains(idLike, "rhel") {
			p.LinuxDistro = DistroFedora
		} else if strings.Contains(idLike, "arch") {
			p.LinuxDistro = DistroArch
		} else if strings.Contains(idLike, "suse") {
			p.LinuxDistro = DistroOpenSUSE
		}
	}
}

func (p *PlatformInfo) detectLinuxPackageManager() {
	switch p.LinuxDistro {
	case DistroDebian, DistroUbuntu:
		if CommandExists("apt-get") {
			p.PackageManager = PkgMgrApt
		}
	case DistroFedora, DistroRHEL, DistroCentOS:
		if CommandExists("dnf") {
			p.PackageManager = PkgMgrDnf
		}
	case DistroArch:
		if CommandExists("pacman") {
			p.PackageManager = PkgMgrPacman
		}
	case DistroAlpine:
		if CommandExists("apk") {
			p.PackageManager = PkgMgrApk
		}
	case DistroOpenSUSE:
		if CommandExists("zypper") {
			p.PackageManager = PkgMgrZypper
		}
	case DistroGentoo:
		if CommandExists("emerge") {
			p.PackageManager = PkgMgrEmerge
		}
	case DistroVoid:
		if CommandExists("xbps-install") {
			p.PackageManager = PkgMgrXbpsInstall
		}
	default:
		p.detectLinuxPackageManagerFallback()
	}
}

func (p *PlatformInfo) detectLinuxPackageManagerFallback() {
	// Try distribution-specific package managers in order of popularity
	if CommandExists("apt-get") {
		p.PackageManager = PkgMgrApt
	} else if CommandExists("dnf") {
		p.PackageManager = PkgMgrDnf
	} else if CommandExists("pacman") {
		p.PackageManager = PkgMgrPacman
	} else if CommandExists("zypper") {
		p.PackageManager = PkgMgrZypper
	} else if CommandExists("apk") {
		p.PackageManager = PkgMgrApk
	} else if CommandExists("emerge") {
		p.PackageManager = PkgMgrEmerge
	} else if CommandExists("xbps-install") {
		p.PackageManager = PkgMgrXbpsInstall
	} else if CommandExists("snap") {
		// Cross-platform package managers as last resort
		p.PackageManager = PkgMgrSnap
	} else if CommandExists("flatpak") {
		p.PackageManager = PkgMgrFlatpak
	}
}

// DetectPlatform detects the current platform information
func DetectPlatform() *PlatformInfo {
	info := &PlatformInfo{
		OS:             OSUnknown,
		Arch:           runtime.GOARCH,
		LinuxDistro:    DistroUnknown,
		LinuxDistroID:  "",
		PackageManager: PkgMgrUnknown,
	}

	switch runtime.GOOS {
	case "linux":
		info.OS = OSLinux
		info.detectLinuxDistro()
		info.detectLinuxPackageManager()
	case "darwin":
		info.OS = OSDarwin
		info.detectDarwinPackageManager()
	case "windows":
		info.OS = OSWindows
		info.detectWindowsPackageManager()
	case "freebsd", "netbsd", "openbsd", "dragonfly":
		info.OS = OSFreeBSD
		info.detectBSDPackageManager()
	}

	return info
}

func (p *PlatformInfo) detectWindowsPackageManager() {
	if CommandExists("winget") {
		p.PackageManager = PkgMgrWinget
	} else if CommandExists("choco") {
		p.PackageManager = PkgMgrChoco
	}
}

// IsRoot checks if the current process has root/administrator privileges
func IsRoot() bool {
	switch runtime.GOOS {
	case "windows":
		return isWindowsAdmin()
	default:
		// Unix-like systems (Linux, macOS, BSD)
		return os.Getuid() == 0
	}
}

// isWindowsAdmin checks for administrator privileges on Windows
func isWindowsAdmin() bool {
	// Try to execute a command that requires admin privileges
	// We use "net session" which fails if not running as admin
	cmd := exec.Command("cmd", "/C", "net", "session")
	err := cmd.Run()
	return err == nil
}

func parseOSRelease(path string) (map[string]string, error) {
	result := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], "\"'")
		result[key] = value
	}

	return result, scanner.Err()
}
