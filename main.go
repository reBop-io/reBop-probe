package main // import "github.com/nicocha/rebopagent"

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/urfave/cli"
)

//const ServiceName = "rebopagent"

// Config ... exported
type Config struct {
	User struct {
		Rebopapikey string `yaml:"rebopapikey", envconfig:"rebop_APIKey"`
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
}

var ext = []string{".cer", ".cert", ".pem", ".der", ".crt"}

var hostname = gethostname()
var ipaddress = getipaddress()
var start = time.Now()
var parsedCount = 0
var validCount = 0
var mutex sync.RWMutex
var mutex2 sync.RWMutex

func main() {
	var cfg Config
	getrebopConfig(&cfg)

	app := cli.NewApp()
	app.Name = "rebopagent"
	app.Version = "0.1.0"
	// Possible command for rebop-agent are
	// scan : scans localhost for certificate
	// send : send local rebop file to remote rebop server
	// acme-cert : get new or renew certificate with ACME PKI (letsencrypt or other)
	app.Usage = "Scan local drives for certificates and send them to reBop.\n\t\tGet and renew certificate from an ACME PKI (LetsEncrypt or other)"
	app.Commands = []cli.Command{
		{
			Name: "scan",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "Scan path from `PATH` and store results ",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "Output file to `FILE`",
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
			Name: "send",
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
					err = rebopSend(certArray, rebopRandomString(5), cfg)
					if err != nil {
						fmt.Println(err)
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
