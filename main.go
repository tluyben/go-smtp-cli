package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Connection settings
	Server       string
	Port         int
	IPv4         bool
	IPv6         bool
	LocalAddr    string
	HelloHost    string
	DisableEHLO  bool
	ForceEHLO    bool

	// TLS/SSL settings
	DisableStartTLS bool
	SSL             bool
	DisableSSL      bool
	SSLCAFile       string
	SSLCAPath       string

	// Authentication
	User         string
	Pass         string
	AuthLogin    bool
	AuthPlain    bool
	AuthCramMD5  bool
	Auth         bool

	// Sender/Recipients
	From string
	To   []string
	Cc   []string
	Bcc  []string

	// Envelope
	MailFrom string
	RcptTo   []string

	// Message content
	Data         string
	Subject      string
	BodyPlain    string
	BodyHTML     string
	Charset      string
	TextEncoding string
	Attach       []string
	AttachInline []string
	AddHeader    []string
	ReplaceHeader []string
	RemoveHeader []string

	// Other
	Verbose         int
	PrintOnly       bool
	MissingModulesOK bool
	Version         bool
	Help            bool
}

type SMTPClient struct {
	conn   net.Conn
	text   *textproto.Conn
	config *Config
	ehlo   bool
	auth   []string
}

const version = "3.10"

func main() {
	config := parseFlags()

	if config.Version {
		fmt.Printf("smtp-cli version %s\n", version)
		return
	}

	if config.Help {
		flag.Usage()
		return
	}

	if err := sendMail(config); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() *Config {
	config := &Config{}

	// Connection flags
	flag.StringVar(&config.Server, "server", "", "Host name or IP address of the SMTP server")
	flag.IntVar(&config.Port, "port", 25, "Port where the SMTP server is listening")
	flag.BoolVar(&config.IPv4, "4", false, "Use standard IP (IPv4) protocol")
	flag.BoolVar(&config.IPv4, "ipv4", false, "Use standard IP (IPv4) protocol")
	flag.BoolVar(&config.IPv6, "6", false, "Use IPv6 protocol")
	flag.BoolVar(&config.IPv6, "ipv6", false, "Use IPv6 protocol")
	flag.StringVar(&config.LocalAddr, "local-addr", "", "Specify local address")
	flag.StringVar(&config.HelloHost, "hello-host", "", "String to use in the EHLO/HELO command")
	flag.BoolVar(&config.DisableEHLO, "disable-ehlo", false, "Don't use ESMTP EHLO command, only HELO")
	flag.BoolVar(&config.ForceEHLO, "force-ehlo", false, "Use EHLO even if server doesn't say ESMTP")

	// TLS/SSL flags
	flag.BoolVar(&config.DisableStartTLS, "disable-starttls", false, "Don't use encryption even if the remote host offers it")
	flag.BoolVar(&config.SSL, "ssl", false, "Start in SMTP/SSL mode (aka SSMTP)")
	flag.BoolVar(&config.DisableSSL, "disable-ssl", false, "Don't start SSMTP even if --port=465")
	flag.StringVar(&config.SSLCAFile, "ssl-ca-file", "", "Verify the server's SSL certificate against a trusted CA root certificate file")
	flag.StringVar(&config.SSLCAPath, "ssl-ca-path", "", "Similar to --ssl-ca-file but will look for the appropriate root certificate file in the given directory")

	// Authentication flags
	flag.StringVar(&config.User, "user", "", "Username for SMTP authentication")
	flag.StringVar(&config.Pass, "pass", "", "Corresponding password")
	flag.BoolVar(&config.AuthLogin, "auth-login", false, "Enable only AUTH LOGIN method")
	flag.BoolVar(&config.AuthPlain, "auth-plain", false, "Enable only AUTH PLAIN method")
	flag.BoolVar(&config.AuthCramMD5, "auth-cram-md5", false, "Enable only AUTH CRAM-MD5 method")
	flag.BoolVar(&config.Auth, "auth", false, "Enable all supported methods")

	// Sender/Recipients flags
	flag.StringVar(&config.From, "from", "", "Sender's name address (or address only)")
	flag.Func("to", "Message recipients", func(s string) error {
		// Handle comma-separated addresses
		addresses := strings.Split(s, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				config.To = append(config.To, addr)
			}
		}
		return nil
	})
	flag.Func("cc", "Message recipients (CC)", func(s string) error {
		// Handle comma-separated addresses
		addresses := strings.Split(s, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				config.Cc = append(config.Cc, addr)
			}
		}
		return nil
	})
	flag.Func("bcc", "Message recipients (BCC)", func(s string) error {
		// Handle comma-separated addresses
		addresses := strings.Split(s, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				config.Bcc = append(config.Bcc, addr)
			}
		}
		return nil
	})

	// Envelope flags
	flag.StringVar(&config.MailFrom, "mail-from", "", "Address to use in MAIL FROM command")
	flag.Func("rcpt-to", "Address to use in RCPT TO command", func(s string) error {
		config.RcptTo = append(config.RcptTo, s)
		return nil
	})

	// Message content flags
	flag.StringVar(&config.Data, "data", "", "Name of file to send after DATA command")
	flag.StringVar(&config.Subject, "subject", "", "Subject of the message")
	flag.StringVar(&config.BodyPlain, "body-plain", "", "Plaintext body of the message")
	flag.StringVar(&config.BodyHTML, "body-html", "", "HTML body of the message")
	flag.StringVar(&config.Charset, "charset", "UTF-8", "Character set used for Subject and Body")
	flag.StringVar(&config.TextEncoding, "text-encoding", "quoted-printable", "Content-Transfer-Encoding for text parts")
	flag.Func("attach", "Attach a given filename", func(s string) error {
		config.Attach = append(config.Attach, s)
		return nil
	})
	flag.Func("attach-inline", "Attach a given filename as inline", func(s string) error {
		config.AttachInline = append(config.AttachInline, s)
		return nil
	})
	flag.Func("add-header", "Add header", func(s string) error {
		config.AddHeader = append(config.AddHeader, s)
		return nil
	})
	flag.Func("replace-header", "Replace header", func(s string) error {
		config.ReplaceHeader = append(config.ReplaceHeader, s)
		return nil
	})
	flag.Func("remove-header", "Remove header", func(s string) error {
		config.RemoveHeader = append(config.RemoveHeader, s)
		return nil
	})

	// Other flags
	flag.IntVar(&config.Verbose, "verbose", 0, "Be more verbose, print the SMTP session")
	flag.BoolVar(&config.PrintOnly, "print-only", false, "Dump the composed MIME message to standard output")
	flag.BoolVar(&config.MissingModulesOK, "missing-modules-ok", false, "Don't complain about missing optional modules")
	flag.BoolVar(&config.Version, "version", false, "Print version")
	flag.BoolVar(&config.Help, "help", false, "Show help")

	flag.Parse()

	// Handle server:port format
	if strings.Contains(config.Server, ":") {
		parts := strings.Split(config.Server, ":")
		config.Server = parts[0]
		if len(parts) > 1 {
			if port, err := strconv.Atoi(parts[1]); err == nil {
				config.Port = port
			}
		}
	}

	// Auto-enable SSL for port 465
	if config.Port == 465 && !config.DisableSSL {
		config.SSL = true
	}

	return config
}

func sendMail(config *Config) error {
	// If no server specified, try to resolve MX records
	if config.Server == "" {
		if len(config.To) == 0 && len(config.Cc) == 0 && len(config.Bcc) == 0 {
			return fmt.Errorf("no server specified and no recipients to resolve MX records")
		}
		// Get domain from first recipient
		var firstRecipient string
		if len(config.To) > 0 {
			firstRecipient = config.To[0]
		} else if len(config.Cc) > 0 {
			firstRecipient = config.Cc[0]
		} else if len(config.Bcc) > 0 {
			firstRecipient = config.Bcc[0]
		}
		
		addr, err := mail.ParseAddress(firstRecipient)
		if err != nil {
			return fmt.Errorf("failed to parse recipient address: %w", err)
		}
		domain := strings.Split(addr.Address, "@")[1]
		
		mxRecords, err := net.LookupMX(domain)
		if err != nil {
			return fmt.Errorf("failed to lookup MX records for %s: %w", domain, err)
		}
		if len(mxRecords) == 0 {
			return fmt.Errorf("no MX records found for %s", domain)
		}
		config.Server = mxRecords[0].Host
		if config.Verbose > 0 {
			fmt.Printf("Resolved MX record: %s\n", config.Server)
		}
	}

	// Create message
	message, err := composeMessage(config)
	if err != nil {
		return fmt.Errorf("failed to compose message: %w", err)
	}

	if config.PrintOnly {
		fmt.Print(message)
		return nil
	}

	// Connect to SMTP server
	client, err := connectSMTP(config)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Send EHLO/HELO
	if err := client.Hello(); err != nil {
		return fmt.Errorf("failed to send HELO/EHLO: %w", err)
	}

	// Start TLS if available
	if !config.DisableStartTLS && !config.SSL {
		if err := client.StartTLS(); err != nil {
			if config.Verbose > 0 {
				fmt.Printf("STARTTLS warning: %v\n", err)
			}
			// If we have credentials and STARTTLS failed, this is likely a problem
			if config.User != "" && client.ehlo {
				// Check if server supports STARTTLS by looking for it in EHLO response
				// If it does but we failed, that's an error
				// For now, we'll continue but the auth will likely fail
			}
		}
	}

	// Authenticate if credentials provided
	if config.User != "" {
		if err := client.Authenticate(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	mailFrom := config.MailFrom
	if mailFrom == "" && config.From != "" {
		addr, err := mail.ParseAddress(config.From)
		if err == nil {
			mailFrom = addr.Address
		} else {
			mailFrom = config.From
		}
	}
	if err := client.MailFrom(mailFrom); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Add recipients
	recipients := []string{}
	if len(config.RcptTo) > 0 {
		recipients = config.RcptTo
	} else {
		// Extract addresses from To, Cc, Bcc
		for _, to := range append(append(config.To, config.Cc...), config.Bcc...) {
			if addr, err := mail.ParseAddress(to); err == nil {
				recipients = append(recipients, addr.Address)
			} else {
				recipients = append(recipients, to)
			}
		}
	}

	for _, rcpt := range recipients {
		if err := client.RcptTo(rcpt); err != nil {
			return fmt.Errorf("RCPT TO %s failed: %w", rcpt, err)
		}
	}

	// Send data
	if err := client.Data(message); err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	return nil
}

func connectSMTP(config *Config) (*SMTPClient, error) {
	network := "tcp"
	if config.IPv4 {
		network = "tcp4"
	} else if config.IPv6 {
		network = "tcp6"
	}

	address := fmt.Sprintf("%s:%d", config.Server, config.Port)
	
	var conn net.Conn
	var err error
	
	if config.LocalAddr != "" {
		localAddr, err := net.ResolveTCPAddr(network, config.LocalAddr)
		if err != nil {
			return nil, err
		}
		dialer := &net.Dialer{
			LocalAddr: localAddr,
			Timeout:   30 * time.Second,
		}
		conn, err = dialer.Dial(network, address)
	} else {
		conn, err = net.DialTimeout(network, address, 30*time.Second)
	}
	
	if err != nil {
		return nil, err
	}

	if config.SSL {
		tlsConfig := &tls.Config{
			ServerName: config.Server,
		}
		if config.SSLCAFile != "" || config.SSLCAPath != "" {
			// TODO: Implement custom CA handling
		}
		conn = tls.Client(conn, tlsConfig)
	}

	client := &SMTPClient{
		conn:   conn,
		text:   textproto.NewConn(conn),
		config: config,
	}

	// Read greeting
	code, msg, err := client.text.ReadResponse(220)
	if config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	if err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (c *SMTPClient) Close() error {
	if c.config.Verbose > 0 {
		fmt.Println("C: QUIT")
	}
	c.text.PrintfLine("QUIT")
	code, msg, _ := c.text.ReadResponse(221)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return c.conn.Close()
}

func (c *SMTPClient) Hello() error {
	hostname := c.config.HelloHost
	if hostname == "" {
		hostname, _ = os.Hostname()
		if hostname == "" {
			hostname = "localhost"
		}
	}

	if !c.config.DisableEHLO {
		if c.config.Verbose > 0 {
			fmt.Printf("C: EHLO %s\n", hostname)
		}
		if err := c.text.PrintfLine("EHLO %s", hostname); err != nil {
			return err
		}
		code, msg, err := c.text.ReadResponse(250)
		if c.config.Verbose > 0 && code > 0 {
			fmt.Printf("S: %d %s\n", code, strings.ReplaceAll(msg, "\n", "\n   "))
		}
		if err == nil {
			c.ehlo = true
			// Parse capabilities
			lines := strings.Split(msg, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "AUTH ") {
					// Extract AUTH methods after "AUTH "
					authIndex := strings.Index(line, "AUTH ")
					if authIndex != -1 {
						authMethods := line[authIndex+5:]
						c.auth = strings.Fields(authMethods)
					}
				}
			}
			return nil
		}
		if code != 502 && code != 500 {
			return err
		}
	}

	// Fall back to HELO
	if c.config.Verbose > 0 {
		fmt.Printf("C: HELO %s\n", hostname)
	}
	if err := c.text.PrintfLine("HELO %s", hostname); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(250)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) StartTLS() error {
	if !c.ehlo {
		return fmt.Errorf("STARTTLS requires EHLO")
	}

	if c.config.Verbose > 0 {
		fmt.Println("C: STARTTLS")
	}
	if err := c.text.PrintfLine("STARTTLS"); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(220)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		ServerName: c.config.Server,
	}
	c.conn = tls.Client(c.conn, tlsConfig)
	c.text = textproto.NewConn(c.conn)

	// Re-send EHLO after STARTTLS
	return c.Hello()
}

func (c *SMTPClient) Authenticate() error {
	if len(c.auth) == 0 {
		return fmt.Errorf("no authentication methods available")
	}

	// Try authentication methods
	for _, method := range c.auth {
		switch strings.ToUpper(method) {
		case "PLAIN":
			if c.config.Auth || c.config.AuthPlain || (!c.config.AuthLogin && !c.config.AuthCramMD5) {
				return c.authPlain()
			}
		case "LOGIN":
			if c.config.Auth || c.config.AuthLogin || (!c.config.AuthPlain && !c.config.AuthCramMD5) {
				return c.authLogin()
			}
		case "CRAM-MD5":
			if c.config.Auth || c.config.AuthCramMD5 || (!c.config.AuthPlain && !c.config.AuthLogin) {
				return c.authCramMD5()
			}
		}
	}

	return fmt.Errorf("no suitable authentication method found")
}

func (c *SMTPClient) authPlain() error {
	auth := fmt.Sprintf("\x00%s\x00%s", c.config.User, c.config.Pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	
	if c.config.Verbose > 0 {
		fmt.Printf("C: AUTH PLAIN [credentials]\n")
	}
	if err := c.text.PrintfLine("AUTH PLAIN %s", encoded); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(235)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) authLogin() error {
	if c.config.Verbose > 0 {
		fmt.Println("C: AUTH LOGIN")
	}
	if err := c.text.PrintfLine("AUTH LOGIN"); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(334)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	if err != nil {
		return err
	}

	// Send username
	if c.config.Verbose > 0 {
		fmt.Printf("C: [username]\n")
	}
	if err := c.text.PrintfLine("%s", base64.StdEncoding.EncodeToString([]byte(c.config.User))); err != nil {
		return err
	}
	code, msg, err = c.text.ReadResponse(334)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	if err != nil {
		return err
	}

	// Send password
	if c.config.Verbose > 0 {
		fmt.Printf("C: [password]\n")
	}
	if err := c.text.PrintfLine("%s", base64.StdEncoding.EncodeToString([]byte(c.config.Pass))); err != nil {
		return err
	}
	code, msg, err = c.text.ReadResponse(235)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) authCramMD5() error {
	if c.config.Verbose > 0 {
		fmt.Println("C: AUTH CRAM-MD5")
	}
	if err := c.text.PrintfLine("AUTH CRAM-MD5"); err != nil {
		return err
	}
	code, challenge, err := c.text.ReadResponse(334)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, challenge)
	}
	if err != nil {
		return err
	}

	// Decode challenge
	decoded, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		return err
	}

	// Calculate response
	h := hmac.New(md5.New, []byte(c.config.Pass))
	h.Write(decoded)
	response := fmt.Sprintf("%s %x", c.config.User, h.Sum(nil))
	
	if c.config.Verbose > 0 {
		fmt.Printf("C: [credentials]\n")
	}
	if err := c.text.PrintfLine("%s", base64.StdEncoding.EncodeToString([]byte(response))); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(235)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) MailFrom(address string) error {
	if c.config.Verbose > 0 {
		fmt.Printf("C: MAIL FROM:<%s>\n", address)
	}
	if err := c.text.PrintfLine("MAIL FROM:<%s>", address); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(250)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) RcptTo(address string) error {
	if c.config.Verbose > 0 {
		fmt.Printf("C: RCPT TO:<%s>\n", address)
	}
	if err := c.text.PrintfLine("RCPT TO:<%s>", address); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(250)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func (c *SMTPClient) Data(message string) error {
	if c.config.Verbose > 0 {
		fmt.Println("C: DATA")
	}
	if err := c.text.PrintfLine("DATA"); err != nil {
		return err
	}
	code, msg, err := c.text.ReadResponse(354)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	if err != nil {
		return err
	}

	// Send message
	if c.config.Verbose > 1 {
		fmt.Printf("C: [Message body, %d bytes]\n", len(message))
	}
	w := c.text.DotWriter()
	if _, err := w.Write([]byte(message)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if c.config.Verbose > 0 {
		fmt.Println("C: .")
	}

	code, msg, err = c.text.ReadResponse(250)
	if c.config.Verbose > 0 && code > 0 {
		fmt.Printf("S: %d %s\n", code, msg)
	}
	return err
}

func composeMessage(config *Config) (string, error) {
	if config.Data != "" {
		// Read complete message from file
		var data []byte
		var err error
		if config.Data == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(config.Data)
		}
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Compose message from components
	var buf strings.Builder
	headers := make(map[string]string)

	// Basic headers
	if config.From != "" {
		headers["From"] = config.From
	}
	if len(config.To) > 0 {
		headers["To"] = strings.Join(config.To, ", ")
	}
	if len(config.Cc) > 0 {
		headers["Cc"] = strings.Join(config.Cc, ", ")
	}
	if config.Subject != "" {
		headers["Subject"] = mime.QEncoding.Encode(config.Charset, config.Subject)
	}
	headers["Date"] = time.Now().Format(time.RFC1123Z)
	headers["Message-ID"] = fmt.Sprintf("<%d.%d@%s>", time.Now().Unix(), os.Getpid(), getHostname())
	headers["MIME-Version"] = "1.0"

	// Apply header modifications
	for _, h := range config.RemoveHeader {
		delete(headers, h)
	}
	for _, h := range config.ReplaceHeader {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Write headers
	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	for _, h := range config.AddHeader {
		buf.WriteString(h + "\r\n")
	}

	// Determine content type and write body
	hasAttachments := len(config.Attach) > 0 || len(config.AttachInline) > 0
	hasMultipleBodyParts := config.BodyPlain != "" && config.BodyHTML != ""

	if hasAttachments || hasMultipleBodyParts {
		// Multipart message
		boundary := fmt.Sprintf("----=_Part_%d_%d", time.Now().Unix(), os.Getpid())
		
		if hasAttachments && hasMultipleBodyParts {
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		} else if hasMultipleBodyParts {
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		} else {
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		}
		buf.WriteString("\r\n")

		// Write body parts
		if config.BodyPlain != "" {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: text/plain; charset=\"%s\"\r\n", config.Charset))
			buf.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n\r\n", config.TextEncoding))
			body, err := readBodyContent(config.BodyPlain)
			if err != nil {
				return "", err
			}
			buf.WriteString(encodeBody(body, config.TextEncoding))
			buf.WriteString("\r\n")
		}

		if config.BodyHTML != "" {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			
			if len(config.AttachInline) > 0 {
				// Multipart/related for inline attachments
				relatedBoundary := fmt.Sprintf("----=_Related_%d_%d", time.Now().Unix(), os.Getpid())
				buf.WriteString(fmt.Sprintf("Content-Type: multipart/related; boundary=\"%s\"\r\n\r\n", relatedBoundary))
				
				buf.WriteString(fmt.Sprintf("--%s\r\n", relatedBoundary))
				buf.WriteString(fmt.Sprintf("Content-Type: text/html; charset=\"%s\"\r\n", config.Charset))
				buf.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n\r\n", config.TextEncoding))
				body, err := readBodyContent(config.BodyHTML)
				if err != nil {
					return "", err
				}
				buf.WriteString(encodeBody(body, config.TextEncoding))
				buf.WriteString("\r\n")
				
				// Add inline attachments
				for _, attachment := range config.AttachInline {
					if err := addAttachment(&buf, attachment, relatedBoundary, true); err != nil {
						return "", err
					}
				}
				
				buf.WriteString(fmt.Sprintf("--%s--\r\n", relatedBoundary))
			} else {
				buf.WriteString(fmt.Sprintf("Content-Type: text/html; charset=\"%s\"\r\n", config.Charset))
				buf.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n\r\n", config.TextEncoding))
				body, err := readBodyContent(config.BodyHTML)
				if err != nil {
					return "", err
				}
				buf.WriteString(encodeBody(body, config.TextEncoding))
				buf.WriteString("\r\n")
			}
		}

		// Add regular attachments
		for _, attachment := range config.Attach {
			if err := addAttachment(&buf, attachment, boundary, false); err != nil {
				return "", err
			}
		}

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Simple message
		if config.BodyHTML != "" {
			buf.WriteString(fmt.Sprintf("Content-Type: text/html; charset=\"%s\"\r\n", config.Charset))
			buf.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n\r\n", config.TextEncoding))
			body, err := readBodyContent(config.BodyHTML)
			if err != nil {
				return "", err
			}
			buf.WriteString(encodeBody(body, config.TextEncoding))
		} else if config.BodyPlain != "" {
			buf.WriteString(fmt.Sprintf("Content-Type: text/plain; charset=\"%s\"\r\n", config.Charset))
			buf.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n\r\n", config.TextEncoding))
			body, err := readBodyContent(config.BodyPlain)
			if err != nil {
				return "", err
			}
			buf.WriteString(encodeBody(body, config.TextEncoding))
		} else {
			buf.WriteString("\r\n")
		}
	}

	return buf.String(), nil
}

func readBodyContent(input string) (string, error) {
	// Check if input is a filename
	if _, err := os.Stat(input); err == nil {
		data, err := os.ReadFile(input)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	// Otherwise treat as literal content
	return input, nil
}

func encodeBody(body, encoding string) string {
	switch encoding {
	case "base64":
		return base64.StdEncoding.EncodeToString([]byte(body))
	case "quoted-printable":
		// Simple quoted-printable encoding
		var buf strings.Builder
		for i, b := range []byte(body) {
			if b == '=' || b < 32 || b > 126 {
				buf.WriteString(fmt.Sprintf("=%02X", b))
			} else {
				buf.WriteByte(b)
			}
			if i > 0 && i%76 == 0 {
				buf.WriteString("=\r\n")
			}
		}
		return buf.String()
	default:
		return body
	}
}

func addAttachment(buf *strings.Builder, attachment, boundary string, inline bool) error {
	// Parse attachment (filename[@mimetype])
	parts := strings.SplitN(attachment, "@", 2)
	filename := parts[0]
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	mimeType := "application/octet-stream"
	if len(parts) > 1 {
		mimeType = parts[1]
	} else {
		// Guess MIME type from extension
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".txt":
			mimeType = "text/plain"
		case ".html", ".htm":
			mimeType = "text/html"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".pdf":
			mimeType = "application/pdf"
		case ".zip":
			mimeType = "application/zip"
		}
	}

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", mimeType, filepath.Base(filename)))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	
	if inline {
		buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", filepath.Base(filename)))
		buf.WriteString("Content-Disposition: inline\r\n")
	} else {
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filepath.Base(filename)))
	}
	
	buf.WriteString("\r\n")
	
	// Encode in base64 with proper line breaks
	encoded := base64.StdEncoding.EncodeToString(data)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
	
	return nil
}

func getHostname() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	return hostname
}