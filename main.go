package main

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/rebop-io/reBop-probe/log"
	"github.com/urfave/cli"
	"github.com/zhexuany/wordGenerator"
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

var ext = []string{".cer", ".cert", ".pem", ".der", ".crt"}

var hostname = gethostname()
var ipaddress = getipaddress()
var start = time.Now()
var parsedCount = 0
var validCount = 0
var knownCount = 0
var errorCount = 0
var mutex sync.RWMutex

// var mutex2 sync.RWMutex
var hashtable = make(map[[32]byte][32]byte)
var app = cli.NewApp()

func main() {
	var cfg Config
	getrebopConfig(&cfg)

	// Get local db
	if err := loadLocalDB(cfg.Probe.Filedb, &hashtable); err != nil {
		log.Infof("No local database found, creating %s", cfg.Probe.Filedb)
	}
	defer saveLocaDB(cfg.Probe.Filedb, hashtable)

	app.Name = "reBop-probe"
	app.Version = "0.6.0"
	// Possible command for rebop-probe are
	// scan : scans localhost for certificate
	// scansend : scans and sends localhost reBop file to remote reBop server
	// acme-cert : get new or renew certificate with ACME PKI (letsencrypt or other)
	app.Usage = "Scan local filesystem for certificates and send them to reBop.\n\t\tGet and renew certificate from an ACME PKI (Let's Encrypt or other)\n\t\tCheck the documentation at https://docs.rebop.io"
	app.Commands = []cli.Command{
		{
			Name: "scan",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "Scan path from `PATH`",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "Output to `FILE`",
				},
			},
			Action: func(c *cli.Context) error {
				length, certArray, err := rebopScan(c.String("path"))
				if err != nil {
					log.Fatal(err)
				}
				if length > 0 {
					f, err := rebopStore(certArray, c.String("out"))
					if err != nil {
						log.Fatal(err)
					}
					log.Infof("reBop file created: %s", f)
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
					log.Fatal(err)
				}
				if lengh > 0 {
					//err = rebopSend(certArray, rebopRandomString(5), cfg)
					uploadName := "reBop-" + wordGenerator.GetWord(5) + ".json"
					err = rebopSend(certArray, uploadName, cfg)
					if err != nil {
						log.Fatal(err)
					}
					log.Infof("reBop file [%s] successfully sent", uploadName)
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
					log.Fatal(err)
				}
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
