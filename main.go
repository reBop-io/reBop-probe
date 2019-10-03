package main // import "github.com/nicocha/rebopagent"

import (
	"errors"
	"fmt"
	"log"
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
	// reset : reset local database
	app.Usage = "scan your filesystem for certificates and encrypt them into one file"
	app.Commands = []cli.Command{
		{
			Name: "scan",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "path to scan",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "output file",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() < 2 {
					return errors.New("usage: scan '<path>' '<output file>'")
				}
				length, certArray, err := rebopScan((c.Args()[0]))
				if err != nil {
					//log.Println(err)
					log.Fatal(err)
				}
				if length > 0 {
					f, err := rebopStore(certArray, c.Args()[1])
					if err != nil {
						log.Fatal(err)
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
					Usage: "path to scan",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return errors.New("usage: send <name>")
				}
				lengh, certArray, err := rebopScan((c.Args()[0]))
				if err != nil {
					log.Fatal(err)
				}
				if lengh > 0 {
					err = rebopSend(certArray, rebopRandomString(5), cfg)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println("reBop file successfully sent")
				}
				return nil
			},
		},
		{
			Name: "renew",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "path to store new certificate",
				},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return errors.New("usage: send <name>")
				}
				err := getCertificatefromACME((c.Args()[0]), cfg)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
