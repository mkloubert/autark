# Autark

A command-line tool to install software from repositories as Docker Compose background services.

## What is Autark?

Autark makes it easy to set up and manage software services on your computer. With simple commands, you can install applications that run in the background using Docker Compose.

**Note:** This project is in early development. More features coming soon!

## Features

- Simple command-line interface
- Cross-platform (Linux, macOS, Windows)
- Installs software as Docker Compose services
- Easy to use for beginners

## Requirements

Before using Autark, make sure you have:

- **Docker** installed and running
- **Docker Compose** installed
- **Git** installed (the installer can install this for you)

## Installation

### Quick Install (Recommended)

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/mkloubert/autark/main/install.sh | sudo sh
```

Or using wget:

```bash
wget -qO- https://raw.githubusercontent.com/mkloubert/autark/main/install.sh | sudo sh
```

**Windows (PowerShell as Administrator):**

```powershell
irm https://raw.githubusercontent.com/mkloubert/autark/main/install.ps1 | iex
```

### What the Installer Does

1. Checks for admin/root permissions
2. Detects your operating system and architecture
3. Installs required tools (git, jq) if missing
4. Downloads the latest Go compiler
5. Clones and builds Autark
6. Installs the binary to your system

### Manual Installation

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/mkloubert/autark.git
cd autark

# Build with Go
go build -o autark .

# Move to a directory in your PATH
sudo mv autark /usr/local/bin/
```

## Usage

After installation, you can use Autark from your terminal:

```bash
# Show help
autark --help

# Show version
autark --version
```

More commands will be added as the project develops.

## Configuration

You can customize the installation using environment variables:

| Variable          | Description                      | Default                                                        |
| ----------------- | -------------------------------- | -------------------------------------------------------------- |
| `AUTARK_REPO_URL` | Git repository URL to clone      | `https://github.com/mkloubert/autark.git`                      |
| `AUTARK_BIN`      | Directory to install the binary  | `/usr/local/bin` (Unix) or `C:\Program Files\autark` (Windows) |
| `AUTARK_PKG_MGR`  | Force a specific package manager | Auto-detected                                                  |

### Supported Package Managers

**Linux:**

- apt (Debian, Ubuntu)
- dnf (Fedora, RHEL)
- pacman (Arch Linux)
- zypper (openSUSE)
- apk (Alpine)
- emerge (Gentoo)
- xbps-install (Void Linux)
- snap
- flatpak

**macOS:**

- brew (Homebrew) - recommended
- port (MacPorts)

**Windows:**

- winget - recommended
- choco (Chocolatey)

## Development

### Prerequisites

- Go 1.25 or later
- Git
- Docker and Docker Compose (for containerized development)

### Local Development with Docker

You can use Docker to develop and test the installation scripts on different Linux distributions.

**Available containers:**

| Service    | Distribution | PowerShell |
| ---------- | ------------ | ---------- |
| `debian11` | Debian 11    | Yes        |
| `debian12` | Debian 12    | Yes        |
| `debian13` | Debian 13    | No         |
| `ubuntu20` | Ubuntu 20.04 | Yes        |
| `ubuntu24` | Ubuntu 24.04 | Yes        |
| `ubuntu26` | Ubuntu 26.04 | No         |

**Start a container:**

```bash
# Start Debian 12 container
docker compose up -d --build debian12

# Or start Ubuntu 24 container
docker compose up -d --build ubuntu24

# Or start all containers
docker compose up -d --build
```

**Open a shell inside the container:**

```bash
# For Debian 12
docker compose exec debian12 bash

# For Ubuntu 24
docker compose exec ubuntu24 bash
```

**Inside the container, you can:**

```bash
# Test the shell installation script
./install.sh

# Test the PowerShell installation script (if available)
pwsh ./install.ps1
```

**Stop the containers:**

```bash
# Stop a specific container
docker compose down debian12

# Stop all containers
docker compose down
```

The project directory is mounted to `/app`, so your local changes are available immediately inside the container.

### Building from Source

```bash
# Clone the repository
git clone https://github.com/mkloubert/autark.git
cd autark

# Download dependencies
go mod download

# Build
go build -o autark .

# Run tests
go test ./...
```

### Project Structure

```
autark/
├── cli/
│   └── app/
│       └── app_context.go    # Application context and stream helpers
├── install.sh                 # Unix installation script
├── install.ps1                # Windows/PowerShell installation script
├── go.mod                     # Go module file
├── main.go                    # Entry point
└── README.md                  # This file
```

## Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** a new branch for your feature (`git checkout -b feature/my-feature`)
3. **Make** your changes
4. **Write** tests for your changes
5. **Run** the tests (`go test ./...`)
6. **Commit** your changes (`git commit -m 'Add my feature'`)
7. **Push** to your branch (`git push origin feature/my-feature`)
8. **Open** a Pull Request

### Code Guidelines

- Follow Go best practices and conventions
- Use the Cobra library patterns for CLI commands
- Write tests for all new commands
- Use English for all code and documentation
- Use the stream helpers from `cli/app/app_context.go` for I/O

## Troubleshooting

### Common Issues

**"Permission denied" error:**

- Make sure you run the installer with `sudo` (Unix) or as Administrator (Windows)

**"Package manager not found" error:**

- Set the `AUTARK_PKG_MGR` variable to your package manager
- Example: `AUTARK_PKG_MGR=apt sudo sh install.sh`

**"Go build failed" error:**

- Make sure you have a stable internet connection
- Try running the installer again

### Getting Help

- Check the [Issues](https://github.com/mkloubert/autark/issues) page
- Open a new issue if you found a bug
- Include your OS, architecture, and error message

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
