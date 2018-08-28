package main

import (
	"fmt"
    "os"
    "net"
    "io/ioutil"
    "path/filepath"
    "encoding/json"
    "strings"
    "bytes"
    b64 "encoding/base64"
)

const CertExt = map[string]string{
    "cer": "cer",
    "crt": "crt",
    "cert": "cert",
    "pem": "pem",
}

type Certificate struct {
		filename string `json:"filaname"`
        path     string `json:"path"`
        privatekeypath    string `json:"privatekeypath"`
        cert string `json:"cert"`
        } 

type Certificates []Certificate

func main() {
    fmt.Println(GetHostInfos())
	fmt.Fprintf(os.Stdout, "%s", ParseHostForCertFiles(".cer"))
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func insertNth(s string, n int) string {
    var buffer bytes.Buffer
    var n_1 = n - 1
    var l_1 = len(s) - 1
    for i,rune := range s {
       buffer.WriteRune(rune)
       if i % n == n_1 && i != l_1  {
          buffer.WriteRune('\n')
       }
    }
    return buffer.String()
}

func GetHostInfos() []string {
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
            fmt.Println(err);
        } 
    }
    return hostinfos
}

func ParseHostForCertFiles(ext string) []byte {
    var certificates = Certificates{}
    var pathS  string = "/Users/nicocha/Projects/"
    
	filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
        var cert string
        var i int = 0
        if !f.IsDir() {
            if filepath.Ext(path) == ext {
                dat, err := ioutil.ReadFile(path)
                check(err)
                if (!strings.Contains(string(dat), ("-----BEGIN CERTIFICATE-----"))) {
                    cert = b64.StdEncoding.EncodeToString(dat)
                    cert = insertNth(cert, 64)
                    cert = "-----BEGIN CERTIFICATE-----" + "\n" + cert + "\n" + "-----END CERTIFICATE-----"
                } else {
                    cert = string(dat)
                }
                fmt.Println(i++)
                certificates = append(certificates, Certificate{filename: f.Name(), path: path, cert: cert })
            }
        }
		return nil
    })

    fmt.Println(certificates)
    b, err := json.Marshal(certificates)
    if err != nil {
        fmt.Println(err)
        return nil
    }
	return b
}