package main

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
const CertExt = map[string] string {
    "cer": "cer",
    "crt": "crt",
    "cert": "cert",
    "pem": "pem",
}
*/

type Certificate struct {
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

type Certificates []Certificate

func main() {

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("	rebop-local <path>\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	} else if _, err := os.Stat(os.Args[1]); os.IsNotExist(err) {
		fmt.Println("path %s doesn't exist", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}
	var path, err = filepath.Abs(os.Args[1])
	check(err)

	//var cert Certificates
	parseHostForCertFiles(path)
	//fmt.Println(cert)
	/*var data, err = json.Marshal(cert)
	if err != nil {
		fmt.Printf("Error: %s", err)
	}*/
	//fmt.Printf("%s\n", data)
	//fmt.Println(cert)
}

func check(e error) {
	if e != nil {
		fmt.Println("Error loading file: ", e.Error())
		//panic(e)
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

func GetHostInfos() []string {
	var hostinfos []string

	var hostname, err = os.Hostname()
	//fmt.Println(hostname)
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

func GetHostName() string {
	var hostname, err = os.Hostname()

	if err != nil {
		hostname = "unknown"
	}
	return hostname
}

func GetIPaddress() []string {
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
			//if v == str {
			return true
		}
	}
	return false
}

func parseHostForCertFiles(pathS string) {
	var certificates Certificates
	//var pathS string = "/Users/nicocha/Projects/"
	//var data string
	//var i int = 0
	ext := []string{".cer", ".cert", ".pem", ".der", ".crt"}
	filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		var cert string
		var hostname = GetHostName()
		var ipaddress []string = GetIPaddress()
		//fmt.Println(stringInSlice("toto.cer", ext))

		if !f.IsDir() {
			if stringInSlice(filepath.Ext(path), ext) {
				//filepath.Ext(path) == ext {
				//fmt.Println(path)
				dat, err := ioutil.ReadFile(path)
				check(err)
				if cap(dat) > 0 {
					if !strings.Contains(string(dat), ("PRIVATE KEY")) && !strings.Contains(string(dat), ("PUBLIC KEY")) && !strings.Contains(string(dat), ("-----BEGIN CERTIFICATE-----")) {
						cert = base64.StdEncoding.EncodeToString(dat)
						cert = insertNth(cert, 64)
						cert = "-----BEGIN CERTIFICATE-----" + "\n" + cert + "\n" + "-----END CERTIFICATE-----"
					} else {
						cert = string(dat)
						block, _ := pem.Decode([]byte(cert))
						if block == nil {
							fmt.Println("failed to parse PEM file: ", cert)
						} else {
							_, err := x509.ParseCertificate(block.Bytes)
							if err != nil {
								fmt.Println("failed to parse certificate: ", err.Error())
							} else {

								//fmt.Println(decodedcert.AuthorityKeyId)

								//i = i + 1
								//fmt.Println(certificates)
								//certificate := Certificate{hostname: "Host", port: "Port", filename: f.Name(), path: path, certificate: cert, date: time.Now().Local().Format("2006-01-02"), probe: "locale"}
								// certificate := Certificate{GetHostName(), "", GetIPaddress(), f.Name(), path, cert, time.Now().Local().Format("2006-01-02"), "locale"}
								certificate := Certificate{hostname, "", ipaddress, f.Name(), path, cert, time.Now().UTC().Format("2006-01-02T15:04:05z"), "local"}
								/*var jsonBlob = []byte(`
								{"hostname": "Host", port: "Port", "filename": f.Name(), "path": path, "certificate": cert}
								`)*/
								//certificate := Certificate{}
								/*err = json.Unmarshal(jsonBlob, &certificate)
								if err != nil {
									// nozzle.printError("opening config file", err.Error())
								}*/

								//fmt.Println(certificate)

								certificates = append(certificates, certificate)
							}
						}
					}

				}
			}
			certificateJson, err := json.Marshal(certificates)
			check(err)
			err = ioutil.WriteFile(time.Now().Local().Format("2006-01-02")+"-"+hostname+"-rebop.json", certificateJson, 0644)
			check(err)
		}
		return nil
	})
	//return certificates
}
