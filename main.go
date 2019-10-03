package main // import "github.com/nicocha/rebopagent"

import (
	"bytes"
	"errors"

	//	"flag"

	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
)

const ServiceName = "rebopagent"

var ext = []string{".cer", ".cert", ".pem", ".der", ".crt"}

type fsEntry struct {
	path string
	f    os.FileInfo
}

var hostname = gethostname()
var ipaddress = getipaddress()
var start = time.Now()
var parsedCount = 0
var validCount = 0
var mutex sync.RWMutex
var mutex2 sync.RWMutex

type certificate struct {
	Hostname  string   `json:"hostname"`
	Port      string   `json:"port"`
	Ipaddress []string `json:"ipaddress"`
	Filename  string   `json:"filename"`
	Path      string   `json:"path"`
	//privatekeypath string `json:"privatekeypath"`
	Certificate string `json:"certificate"`
	Date        string `json:"date"`
	Probe       string `json:"probe"`
}

type certificates []certificate

func main() {
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
				certArray, err := rebopScan((c.Args()[0]))
				if err != nil {
					//log.Println(err)
					log.Fatal(err)
				}
				rebopStore(certArray, c.Args()[1])
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
				certArray, err := rebopScan((c.Args()[0]))
				if err != nil {
					//log.Println(err)
					log.Fatal(err)
				}
				rebopSend(certArray, "test.json")
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func check(e error) {
	if e != nil {
		//fmt.Println("Error loading file: ", e.Error())
		fmt.Errorf(e.Error())
		os.Exit(1)
	}
}

func insertNth(s string, n int) string {
	var buffer bytes.Buffer
	var n1 = n - 1
	var l1 = len(s) - 1
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == n1 && i != l1 {
			buffer.WriteRune('\n')
		}
	}
	return buffer.String()
}

func gethostinfos() []string {
	var hostinfos []string
	var hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	hostinfos = append(hostinfos, hostname)
	ifaces, err := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		// handle err
		for _, addr := range addrs {
			hostinfos = append(hostinfos, addr.String())
		}
		if err != nil {
			fmt.Println(err)
		}
	}
	return hostinfos
}

func gethostname() string {
	var hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return hostname
}

func getipaddress() []string {
	var ipaddress []string
	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		// handle err
		for _, addr := range addrs {
			ipaddress = append(ipaddress, addr.String())
		}
		if err != nil {
			fmt.Println(err)
		}
	}
	return ipaddress
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}
