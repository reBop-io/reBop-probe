package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rebop-io/reBop-probe/log"
	"github.com/urfave/cli"
)

// Config
type Config struct {
	Rebopserver struct {
		Host        string `yaml:"host"`
		Port        string `yaml:"port"`
		Proto       string `yaml:"proto"`
		Rebopapikey string `yaml:"rebopapikey", envconfig:"ReBopAPIKey"`
	} `yaml:"rebopserver"`
	Acme struct {
		Cadirurl  string `yaml:"cadirurl"`
		Solver    string `yaml:"solver"`
		Useremail string `yaml:"useremail"`
		Hostname  string `yaml:"hostname"`
	} `yaml:"acme"`
	Probe struct {
		Filedb string `yaml:"filedb"`
	}
}

var debug = false

var ext = []string{".cer", ".cert", ".pem", ".der", ".crt", ".p12", ".pkcs12", ".jks"}

var hostname = gethostname()
var ipaddress = getipaddress()
var start = time.Now()
var parsedCount = int64(0)
var validCount = int64(0)
var knownCount = int64(0)
var errorCount = int64(0)
var mutex sync.RWMutex

var hashtable = make(map[[32]byte][32]byte)
var app = cli.NewApp()

func main() {
	var cfg Config
	getrebopConfig(&cfg)

	// Get local db
	if err := loadLocalDB(cfg.Probe.Filedb, &hashtable); err != nil {
		log.Printf("No local database found, creating %s", cfg.Probe.Filedb)
	}
	defer saveLocaDB(cfg.Probe.Filedb, hashtable)

	app.Name = "reBop-probe"
	app.Version = "1.0.1"
	app.Usage = "Certificate discovery and management tool\n\t\n\tScan filesystem for SSL/TLS certificates, manage ACME certificates, and integrate\n\twith the reBop certificate management platform.\n\n\tDocumentation: https://docs.rebop.io"

	app.Commands = []cli.Command{
		{
			Name:        "scan",
			Usage:       "Scan filesystem for SSL/TLS certificates",
			Description: `Scan a directory recursively for SSL/TLS certificates and generate a reBop\n   report file. The probe will automatically detect certificate files and\n   extract relevant information.`,
			ArgsUsage:   "[PATH]",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Value: ".",
					Usage: "Directory path to scan for certificates",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "Output report to `FILE` (default: rebop_<timestamp>.gz)",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "Enable verbose output",
				},
			},
			Action: func(c *cli.Context) error {
				targetPath := c.String("path")
				if targetPath == "" {
					targetPath = "."
				}

				log.Printf("Starting certificate scan in: %s", targetPath)
				length, certArray, err := rebopScan(targetPath)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Scan failed: %v", err), 1)
				}

				if length == 0 {
					log.Println("No certificates found in the specified path")
					return nil
				}

				outputFile := c.String("out")
				f, err := rebopStore(certArray, outputFile)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Failed to save results: %v", err), 1)
				}

				log.Printf("Scan completed. Results saved to: %s", f)
				log.Printf("Found %d certificates in %d scanned files", length, parsedCount)
				return nil
			},
		},
		{
			Name:      "scansend",
			Usage:     "Scan for certificates and upload to reBop server",
			ArgsUsage: "[PATH]",
			Description: `Scan a directory for SSL/TLS certificates and automatically upload
   the results to the configured reBop server. Requires API credentials to be
   configured in the reBop configuration file.`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Value: "/",
					Usage: "Directory path to scan for certificates",
				},
				cli.StringFlag{
					Name:  "server, s",
					Usage: "reBop server URL (overrides config)",
				},
				cli.BoolFlag{
					Name:  "no-delete",
					Usage: "Keep the generated report file after upload",
				},
			},
			Action: func(c *cli.Context) error {
				targetPath := c.String("path")
				if targetPath == "" {
					targetPath = "/"
				}

				log.Printf("Starting certificate scan in: %s", targetPath)
				length, certArray, err := rebopScan(targetPath)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Scan failed: %v", err), 1)
				}

				if length == 0 {
					log.Println("No certificates found to upload")
					return nil
				}

				tempFile := fmt.Sprintf("rebop_%s.gz", time.Now().Format("20060102-150405"))
				f, err := rebopStore(certArray, tempFile)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Failed to save scan results: %v", err), 1)
				}

				log.Printf("Found %d certificates. Uploading to reBop server...", length)
				certData, err := os.ReadFile(f)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Failed to read certificate file: %v", err), 1)
				}

				uploadName := fmt.Sprintf("scan_%s.gz", time.Now().Format("20060102_150405"))
				if err := rebopSend(certData, uploadName, cfg); err != nil {
					if !c.Bool("no-delete") {
						os.Remove(f) // Clean up temp file on error
					}
					return cli.NewExitError(fmt.Sprintf("Upload failed: %v", err), 1)
				}

				log.Println("Successfully uploaded certificate data to reBop server")
				if !c.Bool("no-delete") {
					os.Remove(f) // Clean up temp file
				}
				return nil
			},
		},
		{
			Name:      "acme-cert",
			Usage:     "Request or renew certificates using ACME protocol",
			ArgsUsage: "[DOMAIN...]",
			Description: `Request new or renew existing SSL/TLS certificates using the ACME protocol
   (e.g., Let's Encrypt). Supports HTTP-01 and DNS-01 challenges.`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "Path to store the certificate files",
					Value: "./certs",
				},
				cli.StringFlag{
					Name:  "email, e",
					Usage: "Email address for account registration and recovery",
				},
				cli.StringFlag{
					Name:  "webroot, w",
					Usage: "Webroot directory for HTTP-01 challenge",
				},
				cli.StringFlag{
					Name:  "dns, d",
					Usage: "DNS provider for DNS-01 challenge (e.g., cloudflare, route53)",
				},
				cli.BoolFlag{
					Name:  "staging",
					Usage: "Use Let's Encrypt staging environment",
				},
				cli.BoolFlag{
					Name:  "force-renew",
					Usage: "Force renewal even if certificate is not expired",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return errors.New("usage: acme-cert [DOMAIN...]")
				}

				domains := c.Args()
				path := c.String("path")
				email := c.String("email")
				// webroot and dns are currently not used but kept for future implementation
				_ = c.String("webroot")
				_ = c.String("dns")
				_ = c.Bool("force-renew") // Not currently used

				// Use staging flag if needed
				if c.Bool("staging") {
					log.Println("Using Let's Encrypt staging environment")
				}

				if email == "" {
					return errors.New("email address is required")
				}

				// Create output directory if it doesn't exist
				if err := os.MkdirAll(path, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %v", err)
				}

				log.Printf("Requesting certificate for domains: %v", domains)
				log.Printf("Using Let's Encrypt %s environment", map[bool]string{true: "Staging", false: "Production"}[c.Bool("staging")])

				// Call the ACME function with the appropriate parameters
				if email == "" {
					return errors.New("email address is required (use --email flag)")
				}

				// Update config with email
				cfg.Acme.Useremail = email

				// The getCertificatefromACME function handles its own file operations
				log.Printf("Requesting certificate for domains: %v", domains)
				if len(domains) == 0 {
					return fmt.Errorf("at least one domain is required")
				}

				// The function handles its own file operations and logging
				if err := getCertificatefromACME(path, cfg); err != nil {
					return fmt.Errorf("failed to obtain certificate: %v", err)
				}

				log.Println("Certificate successfully obtained/renewed")
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
