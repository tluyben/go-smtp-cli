# smtp-cli

A command-line SMTP client written in Go that allows sending emails with various options including authentication, TLS/SSL support, attachments, and more.

## Quick Install

```bash
# macOS Intel
curl -L https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-darwin-amd64 -o smtp-cli && chmod 755 smtp-cli

# macOS Apple Silicon
curl -L https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-darwin-arm64 -o smtp-cli && chmod 755 smtp-cli

# Linux Intel/AMD64
wget https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-linux-amd64 -O smtp-cli && chmod 755 smtp-cli

# Linux ARM64
wget https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-linux-arm64 -O smtp-cli && chmod 755 smtp-cli
```

## Installation

### Download Pre-built Binaries

Download the appropriate binary for your platform:

#### macOS Intel
```bash
curl -L https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-darwin-amd64 -o smtp-cli
chmod 755 smtp-cli
```

#### macOS Apple Silicon (M1/M2/M3)
```bash
curl -L https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-darwin-arm64 -o smtp-cli
chmod 755 smtp-cli
```

#### Linux Intel/AMD64
```bash
wget https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-linux-amd64 -O smtp-cli
chmod 755 smtp-cli
```

#### Linux ARM64
```bash
wget https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-linux-arm64 -O smtp-cli
chmod 755 smtp-cli
```

#### Windows Intel/AMD64
```powershell
# PowerShell
Invoke-WebRequest -Uri https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-windows-amd64.exe -OutFile smtp-cli.exe
```

#### Windows ARM64
```powershell
# PowerShell
Invoke-WebRequest -Uri https://github.com/tluyben/go-smtp-cli/raw/main/release/smtp-cli-windows-arm64.exe -OutFile smtp-cli.exe
```

### Build From Source

1. Install Go (1.20 or later)
2. Clone the repository:
   ```bash
   git clone https://github.com/tluyben/go-smtp-cli.git
   cd go-smtp-cli
   ```
3. Build the binary:
   ```bash
   go build -o smtp-cli main.go
   ```

### Cross-compilation

Use the included Makefile to build for multiple platforms:

```bash
# Build for all platforms
make all

# Build for current platform only
make build

# Create release directory with all binaries
make release

# Create compressed archives
make compress

# Clean build artifacts
make clean

# List built binaries
make list

# See all available targets
make help
```

#### Individual Platform Builds

```bash
# Build specific platforms
make build-windows-amd64   # Windows Intel/AMD64
make build-windows-arm64   # Windows ARM64
make build-darwin-amd64    # macOS Intel
make build-darwin-arm64    # macOS Apple Silicon
make build-linux-amd64     # Linux Intel/AMD64
make build-linux-arm64     # Linux ARM64
```

The following binaries will be created:
- `smtp-cli-windows-amd64.exe` - Windows Intel/AMD64
- `smtp-cli-windows-arm64.exe` - Windows ARM64
- `smtp-cli-darwin-amd64` - macOS Intel
- `smtp-cli-darwin-arm64` - macOS Apple Silicon (M1/M2/M3)
- `smtp-cli-linux-amd64` - Linux Intel/AMD64
- `smtp-cli-linux-arm64` - Linux ARM64 (including Raspberry Pi 64-bit)

## Usage

```bash
./smtp-cli [--options]
```

### Basic Example

```bash
./smtp-cli \
  --server smtp.example.com \
  --port 587 \
  --user myusername \
  --pass mypassword \
  --from sender@example.com \
  --to recipient@example.com \
  --subject "Test Email" \
  --body-plain "This is a test email."
```

### SSL/TLS Example

```bash
./smtp-cli \
  --server smtp.example.com \
  --port 465 \
  --ssl \
  --user myusername \
  --pass mypassword \
  --from sender@example.com \
  --to recipient@example.com \
  --subject "Secure Email" \
  --body-plain "This email is sent over SSL."
```

### Complete Example

```bash
./smtp-cli \
  --server smtp.example.com \
  --port 465 \
  --ssl \
  --user myusername \
  --pass mypassword \
  --from sender@example.com \
  --to recipient@example.com \
  --cc cc-recipient@example.com \
  --bcc bcc1@example.com,bcc2@example.com \
  --subject "Meeting Discussion" \
  --body-plain "Hello,\n\nLet's discuss our upcoming meeting."
```

## Options

### Connection Options
- `--server=<hostname>[:<port>]` - SMTP server hostname or IP address
- `--port=<number>` - Port number (default: 25)
- `-4` or `--ipv4` - Use IPv4 protocol
- `-6` or `--ipv6` - Use IPv6 protocol
- `--local-addr=<address>` - Specify local address
- `--hello-host=<string>` - String to use in EHLO/HELO command
- `--disable-ehlo` - Don't use ESMTP EHLO command, only HELO
- `--force-ehlo` - Use EHLO even if server doesn't say ESMTP

### TLS/SSL Options
- `--disable-starttls` - Don't use encryption even if offered
- `--ssl` - Start in SMTP/SSL mode (default for port 465)
- `--disable-ssl` - Don't start SSMTP even if --port=465
- `--ssl-ca-file=<filename>` - Verify server certificate against CA file
- `--ssl-ca-path=<dirname>` - Directory with CA certificates

### Authentication Options
- `--user=<username>` - Username for SMTP authentication
- `--pass=<password>` - Corresponding password
- `--auth-login` - Enable only AUTH LOGIN method
- `--auth-plain` - Enable only AUTH PLAIN method
- `--auth-cram-md5` - Enable only AUTH CRAM-MD5 method
- `--auth` - Enable all supported methods

### Sender/Recipients
- `--from="Display Name <add@re.ss>"` - Sender's name and address
- `--to="Display Name <add@re.ss>"` - Recipient (can be used multiple times)
- `--cc="Display Name <add@re.ss>"` - CC recipient (can be used multiple times)
- `--bcc="Display Name <add@re.ss>"` - BCC recipient (can be used multiple times)

### Envelope Options (Advanced)
- `--mail-from=<address>` - Address for MAIL FROM command
- `--rcpt-to=<address>` - Address for RCPT TO command (can be used multiple times)

### Message Content
- `--data=<filename>` - Send complete RFC822 message from file (use "-" for stdin)
- `--subject=<subject>` - Subject of the message
- `--body-plain=<text|filename>` - Plain text body
- `--body-html=<text|filename>` - HTML body
- `--charset=<charset>` - Character set (default: UTF-8)
- `--text-encoding=<encoding>` - Content-Transfer-Encoding (7bit, 8bit, binary, base64, quoted-printable)
- `--attach=<filename>[@<MIME/Type>]` - Attach file (can be used multiple times)
- `--attach-inline=<filename>[@<MIME/Type>]` - Attach inline file (can be used multiple times)
- `--add-header="Header: value"` - Add custom header
- `--replace-header="Header: value"` - Replace header
- `--remove-header="Header"` - Remove header

### Other Options
- `--verbose[=<number>]` - Be more verbose, print SMTP session
- `--print-only` - Dump composed message to stdout without sending
- `--missing-modules-ok` - Don't complain about missing optional modules
- `--version` - Print version
- `--help` - Show help

## Features

- **Multiple Recipients**: Support for To, CC, and BCC recipients
- **Authentication**: Supports LOGIN, PLAIN, and CRAM-MD5 authentication methods
- **Encryption**: TLS/STARTTLS and SSL support
- **Attachments**: File attachments with MIME type detection
- **Inline Attachments**: For embedding images in HTML emails
- **Custom Headers**: Add, replace, or remove email headers
- **Multipart Messages**: Support for plain text and HTML bodies
- **DNS MX Lookup**: Automatically resolve SMTP server from recipient's domain
- **Verbose Mode**: Debug SMTP communication
- **Message Preview**: Print composed message without sending

## Version

This is smtp-cli version 3.10, compatible with the original Perl smtp-cli.