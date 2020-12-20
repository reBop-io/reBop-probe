package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type rebopCertificate struct {
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

type rebopCertificates []rebopCertificate

type fsEntry struct {
	path string
	f    os.FileInfo
}

func rebopScan(rootPath string) (int, []byte, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		//fmt.Printf("Couldn't open %s", rootPath)
		return 0, nil, err
	}
	var wg sync.WaitGroup

	paths := make(chan fsEntry, 1)
	errs := make(chan error, 1)
	certs := make(chan *rebopCertificate, 1)
	done := make(chan bool, 1)

	wg.Add(1)
	go parseHostForCertFiles(rootPath, paths, errs, &wg)
	go certWorker(paths, errs, certs, &wg)
	go func() {
		wg.Wait()
		done <- true
	}()

	rebopCertificates := make(rebopCertificates, 0)

	for {
		select {
		case <-done:
			mutex.Lock()
			certificateJSON, err := json.Marshal(rebopCertificates)
			lengh := len(rebopCertificates)
			if err != nil {
				return 0, nil, err
			}
			fmt.Println("reBop scan Completed in :", time.Since(start), "\nParsed", parsedCount, "files\nFound", validCount, "new files,", knownCount, "known files and", errorCount, "files without certificate")
			mutex.Unlock()
			return lengh, certificateJSON, nil
		case cert := <-certs:
			rebopCertificates = append(rebopCertificates, *cert)
		case err := <-errs:
			fmt.Println("error: ", err)
		}
	}
}

func parseHostForCertFiles(pathS string, paths chan fsEntry, errs chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.Walk(pathS, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			errs <- err
			return nil
		}
		if !f.IsDir() {
			absolutePath, _ := filepath.Abs(path)
			paths <- fsEntry{path: absolutePath, f: f}
		}
		return nil
	})
}

func certWorker(entries chan fsEntry, errs chan error, certs chan *rebopCertificate, wg *sync.WaitGroup) {
	for entry := range entries {
		wg.Add(1)
		// too fast, need to save entry value before executing go routine
		entryCopy := entry
		go func() {
			defer wg.Done()
			// FIXME: use a limited pool
			cert, err := parseEntry(entryCopy)
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

func parseEntry(entry fsEntry) (*rebopCertificate, error) {
	var cert string

	// todo: give the task to a worker pool
	//fmt.Println(filepath.Ext(entry.path))
	if stringInSlice(filepath.Ext(entry.path), ext) {
		mutex.Lock()
		parsedCount++
		mutex.Unlock()

		dat, err := ioutil.ReadFile(entry.path)
		if err != nil {
			return nil, err
		}
		if cap(dat) > 0 {
			//fmt.Println("CAP")
			if strings.Contains(string(dat), ("PRIVATE KEY")) || strings.Contains(string(dat), ("PUBLIC KEY")) {
				mutex.Lock()
				errorCount++
				mutex.Unlock()
				return nil, nil
			} else if !strings.Contains(string(dat), ("-----BEGIN CERTIFICATE-----")) {
				cert = base64.StdEncoding.EncodeToString(dat)
				cert = insertNth(cert, 64)
				cert = "-----BEGIN CERTIFICATE-----" + "\n" + cert + "\n" + "-----END CERTIFICATE-----"
			} else {
				//fmt.Println("ELSE")
				cert = string(dat)
				block, _ := pem.Decode([]byte(cert))
				if block == nil {
					mutex.Lock()
					errorCount++
					mutex.Unlock()
					fmt.Println("failed to parse PEM file: ", entry.path)
				} else {
					//fmt.Println(block.Bytes)
					_, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						if strings.Contains(err.Error(), "named curve") {
							fmt.Println(err.Error())
						} else {
							mutex.Lock()
							errorCount++
							mutex.Unlock()
							return nil, fmt.Errorf(err.Error(), entry.path)
						}
					}
					pathhash := sha256.Sum256([]byte(entry.path))
					datahash := sha256.Sum256([]byte(dat))
					if val, ok := hashtable[pathhash]; ok && val == datahash {
						// Exists and value are the same
						//fmt.Printf("pathhash %x exists\n", pathhash)
						mutex.Lock()
						knownCount++
						mutex.Unlock()
						return nil, nil
					}
					//fmt.Println("AFTER RETURN")
					hashtable[pathhash] = datahash
					rebopCertificate := rebopCertificate{
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
					return &rebopCertificate, nil
				}
			}
		}
	}
	return nil, nil
}
