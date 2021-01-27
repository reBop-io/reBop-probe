package main // import "github.com/nicocha/rebopagent"

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/urfave/cli"
	"github.com/zhexuany/wordGenerator"
)

//const ServiceName = "rebopagent"

// Config ... exported
type Config struct {
	User struct {
		ReBopAPIKey string `yaml:"rebopapikey", envconfig:"ReBopAPIKey"`
	} `yaml:"user"`
	Rebopserver struct {
		Host  string `yaml:"host"`
		Port  string `yaml:"port"`
		Proto string `yaml:"proto"`
	} `yaml:"rebopserver"`
	Acme struct {
		Cadirurl  string `yaml:"cadirurl"`
		Useremail string `yaml:"useremail"`
		Hostname  string `yaml:"hostname"`
	} `yaml:"acme"`
	Agent struct {
		Filedb string `yaml:"filedb"`
	}
}

var debug = false

var ext = []string{".cer", ".cert", ".pem", ".der", ".crt"}

var hostname = gethostname()
var ipaddress = getipaddress()
var start = time.Now()
var parsedCount = 0
var validCount = 0
var knownCount = 0
var errorCount = 0
var mutex sync.RWMutex
var mutex2 sync.RWMutex
var hashtable = make(map[[32]byte][32]byte)
var app = cli.NewApp()

func main() {
	var cfg Config
	getrebopConfig(&cfg)

	// Get local db
	if err := loadLocalDB(cfg.Agent.Filedb, &hashtable); err != nil {
		//log.Fatalln(err)
	}
	defer saveLocaDB(cfg.Agent.Filedb, hashtable)

	app.Name = "reBop-agent"
	app.Version = "0.2.0"
	// Possible command for rebop-agent are
	// scan : scans localhost for certificate
	// scansend : scans and sends localhost reBop file to remote reBop server
	// acme-cert : get new or renew certificate with ACME PKI (letsencrypt or other)
	app.Usage = "Scan local drives for certificates and send them to reBop.\n\t\tGet and renew certificate from an ACME PKI (LetsEncrypt or other)"
	app.Commands = []cli.Command{
		{
			Name: "scan",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "Scan path from `PATH` and store results",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "Output to `FILE`",
				},
			},
			Action: func(c *cli.Context) error {
				length, certArray, err := rebopScan(c.String("path"))
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if length > 0 {
					f, err := rebopStore(certArray, c.String("out"))
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					fmt.Println("reBop file created: ", f)
				}
				return nil
			},
		},
		{
			Name: "scansend",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "Scans path from `PATH` and sends result to reBop server",
				},
			},
			Action: func(c *cli.Context) error {
				lengh, certArray, err := rebopScan(c.String("path"))
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if lengh > 0 {
					//err = rebopSend(certArray, rebopRandomString(5), cfg)
					err = rebopSend(certArray, "reBop-"+wordGenerator.GetWord(5), cfg)
					if err != nil {
						fmt.Println(err)
						// Need to ask the user if the created file shall be saved for later
						os.Exit(1)
					}
					fmt.Println("reBop file successfully sent")
				}
				return nil
			},
		},
		{
			Name: "acme-cert",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "path to store new certificate",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return errors.New("usage: renew <path>")
				}
				err := getCertificatefromACME((c.Args()[0]), cfg)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
