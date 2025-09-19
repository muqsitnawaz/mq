# Installation Guide

## Prerequisites

Before installing Engram, ensure you have the following prerequisites:

### System Requirements

* Operating System:
  * Linux (Ubuntu 20.04+, CentOS 8+, Debian 10+)
  * macOS (11.0 Big Sur or later)
  * Windows (10 version 1909+, Windows 11)

* Hardware:
  * Minimum 8GB RAM (16GB recommended)
  * 10GB available disk space
  * 64-bit processor

### Required Software

1. **Go** (version 1.21 or higher)
2. **Git** (version 2.25 or higher)
3. **Docker** (optional, for containerized deployment)
4. **PostgreSQL** (version 14+) or **SQLite** for development

## Installation Methods

### Method 1: Using Go Install (Recommended)

The simplest way to install Engram is using Go's built-in package manager:

```bash
go install github.com/engram/mq@latest
```

> **Note:** This method automatically handles dependencies and places the binary in your `$GOPATH/bin`.

### Method 2: Building from Source

For developers who want to contribute or customize:

1. Clone the repository:
   ```bash
   git clone https://github.com/engram/engram.git
   cd engram
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the binary:
   ```bash
   go build -o mq cmd/mq/main.go
   ```

4. Install to system path:
   ```bash
   sudo mv mq /usr/local/bin/
   ```

### Method 3: Using Docker

For containerized environments:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mq cmd/mq/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/mq /usr/local/bin/
CMD ["mq"]
```

Build and run:
```bash
docker build -t engram:latest .
docker run -it engram:latest mq --help
```

### Method 4: Package Managers

#### Homebrew (macOS/Linux)

- [ ] Tap the repository
- [ ] Update Homebrew
- [ ] Install the package
- [ ] Verify installation

```bash
brew tap engram/tap
brew update
brew install mq
```

#### APT (Debian/Ubuntu)

- [x] Add GPG key
- [x] Add repository
- [ ] Update package list
- [ ] Install package

```bash
curl -fsSL https://packages.engram.ai/gpg | sudo apt-key add -
echo "deb https://packages.engram.ai/apt stable main" | sudo tee /etc/apt/sources.list.d/engram.list
sudo apt update
sudo apt install engram-mq
```

#### YUM/DNF (RHEL/CentOS/Fedora)

```bash
sudo dnf config-manager --add-repo https://packages.engram.ai/rpm/engram.repo
sudo dnf install engram-mq
```

## Post-Installation Setup

### Environment Configuration

Create a configuration file at `~/.engram/config.yaml`:

```yaml
# Basic configuration
cache:
  enabled: true
  size: 100MB
  ttl: 24h

# Semantic search settings
embeddings:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: text-embedding-ada-002

# File watching
watch:
  enabled: true
  patterns:
    - "*.md"
    - "*.markdown"
  ignore:
    - "node_modules/**"
    - ".git/**"
```

### Verification Steps

After installation, verify everything is working:

- [x] Check version: `mq --version`
- [x] Run help: `mq --help`
- [ ] Test basic query: `mq '.headings' README.md`
- [ ] Verify cache: `mq cache status`

### Shell Completion

#### Bash

```bash
echo 'source <(mq completion bash)' >> ~/.bashrc
source ~/.bashrc
```

#### Zsh

```bash
echo 'source <(mq completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

#### Fish

```bash
mq completion fish > ~/.config/fish/completions/mq.fish
```

## Database Setup

### PostgreSQL Configuration

1. Create database and user:
   ```sql
   CREATE DATABASE engram;
   CREATE USER engram_user WITH ENCRYPTED PASSWORD 'secure_password';
   GRANT ALL PRIVILEGES ON DATABASE engram TO engram_user;
   ```

2. Run migrations:
   ```bash
   mq migrate up
   ```

### SQLite Configuration (Development)

For development environments, SQLite is configured automatically:

```bash
export ENGRAM_DB="sqlite://./engram.db"
mq migrate up
```

## Troubleshooting Installation

### Common Issues

#### Issue: Command not found

**Solution:** Add Go's bin directory to your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Issue: Permission denied

**Solution:** Use sudo for system-wide installation:
```bash
sudo go install github.com/engram/mq@latest
```

#### Issue: Dependency conflicts

**Solution:** Clear the module cache and reinstall:
```bash
go clean -modcache
go mod download
```

### Platform-Specific Issues

> **Windows Users:**
> - Enable developer mode for symlink support
> - Use PowerShell as Administrator
> - Install Windows Terminal for better experience

> **macOS Users:**
> - Xcode Command Line Tools required: `xcode-select --install`
> - For M1/M2 Macs, use native ARM64 builds

> **Linux Users:**
> - Ensure glibc version 2.31 or higher
> - For WSL2, enable systemd support

## Development Setup

### IDE Configuration

#### VS Code

Install recommended extensions:
- Go extension by Google
- Markdown All in One
- YAML

Settings (`settings.json`):
```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"]
}
```

#### GoLand/IntelliJ

1. Open project root
2. Configure GOPATH
3. Enable Go modules
4. Set up file watchers

### Testing Installation

Run the test suite to verify your installation:

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Benchmark tests
go test -bench=. ./...
```

Expected output:
```
PASS
ok  	github.com/engram/mq/parser	0.123s
ok  	github.com/engram/mq/engine	0.456s
ok  	github.com/engram/mq/cache	0.789s
```

## Uninstallation

### Remove Binary

```bash
# If installed with go install
rm $(go env GOPATH)/bin/mq

# If installed to /usr/local/bin
sudo rm /usr/local/bin/mq
```

### Remove Configuration

```bash
rm -rf ~/.engram
```

### Remove Cache

```bash
rm -rf ~/.cache/engram
```

## Next Steps

After successful installation:

1. **Read the Tutorial** - Learn basic mq syntax
2. **Explore Examples** - Check the `examples/` directory
3. **Join Community** - Discord server for support
4. **Configure IDE** - Set up your development environment
5. **Try Queries** - Start with simple queries on your markdown files

## Getting Help

If you encounter issues:

* Check the [FAQ](./faq.md)
* Visit our [GitHub Issues](https://github.com/engram/engram/issues)
* Join our [Discord Community](https://discord.gg/engram)
* Email support: support@engram.ai

---

*Installation guide version: 1.0.0 | Last updated: January 2024*