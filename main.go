package rebopAgent // import "github.com/nicocha/rebopAgent"

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"

	//	"flag"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
)

const ServiceName = "rebop-agent"

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
	app.Name = "rebop"
	app.Version = "0.1.0"
	// Possible command for rebop-agent are
	// scan : scans localhost for certificate and save them locally
	// send : scan & send certificate ile to remote rebop server
	// reset : reset local database
	app.Usage = "Ccan your filesystem for certificates and either send them to rebop or save them locally"
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
				rebopfile = scan(c.Args()[0], c.Args()[1])
		},
		{
			Name: "send",
			Action: func(c *cli.Context) error {
				if c.NArg() < 1 {
					return errors.New("usage: send <name>")
				}
				log.Println("in send")
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

func parseHostForCertFiles(pathS string, paths chan fsEntry, errs chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	//certpool, _ := x509.SystemCertPool()
	//fmt.Println(certpool.Subjects())

	//var pathS string = "/Users/nicocha/Projects/"
	//var data string
	var i int = 0

	filepath.Walk(pathS, func(path string, f os.FileInfo, err error) error {
		i++
		if err != nil {
			/*
				for i := 0; i < 100; i++ {
					log.Println("**********************************************")
				}
				log.Println(err)
				log.Println("**********************************************")
			*/
			errs <- err
			return nil
		}
		if !f.IsDir() {
			paths <- fsEntry{path: path, f: f}
		}
		/*
			if i%100 == 0 {
				fmt.Println(i, len(paths), time.Since(start), parsedCount, validCount)
			}
		*/
		return nil
	})
}

func worker(entries chan fsEntry, errs chan error, certs chan *certificate, wg *sync.WaitGroup) {
	for entry := range entries {
		wg.Add(1)
		// too fast, need to save entry value before executing fo routine
		entryCopy := entry
		go func() {
			defer wg.Done()
			// FIXME: use a limited pool
			cert, err := once(entryCopy)
			if err != nil {
				errs <- err
				return
			}
			if cert != nil {
				certs <- cert
			}
		}()
	}
}

func once(entry fsEntry) (*certificate, error) {
	var cert string

	// todo: give the task to a worker pool
	fmt.Println(filepath.Ext(entry.path))
	if stringInSlice(filepath.Ext(entry.path), ext) {
		mutex.Lock()
		parsedCount++
		mutex.Unlock()

		dat, err := ioutil.ReadFile(entry.path)
		if err != nil {
			return nil, err
		}
		//fmt.Println(filepath.Ext(entry.path))
		if cap(dat) > 0 {
			//fmt.Println("CAP")
			if !strings.Contains(string(dat), ("PRIVATE KEY")) && !strings.Contains(string(dat), ("PUBLIC KEY")) && !strings.Contains(string(dat), ("-----BEGIN CERTIFICATE-----")) {
				cert = base64.StdEncoding.EncodeToString(dat)
				cert = insertNth(cert, 64)
				cert = "-----BEGIN CERTIFICATE-----" + "\n" + cert + "\n" + "-----END CERTIFICATE-----"
			} else {
				//fmt.Println("ELSE")
				cert = string(dat)
				block, _ := pem.Decode([]byte(cert))
				if block == nil {
					fmt.Println("failed to parse PEM file: ", entry.path)
				} else {
					//fmt.Println(block.Bytes)
					_, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						if strings.Contains(err.Error(), "named curve") {
							fmt.Println(err.Error())
						} else {
							return nil, fmt.Errorf(err.Error(), entry.path)
						}
					}
					//fmt.Println("AFTER RETURN")
					certificate := certificate{
						hostname,
						"",
						ipaddress,
						entry.f.Name(),
						entry.path,
						cert,
						time.Now().UTC().Format("2006-01-02T15:04:05z"),
						"local",
					}
					mutex.Lock()
					validCount++
					mutex.Unlock()
					return &certificate, nil
				}
			}
		}
	}
	return nil, nil
}
