package main

import (
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

func rebopScan(rootPath string) ([]byte, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		fmt.Printf("Couldn't open %s", rootPath)
		return nil, err
	}
	var wg sync.WaitGroup

	paths := make(chan fsEntry, 1)
	errs := make(chan error, 1)
	certs := make(chan *certificate, 1)
	done := make(chan bool, 1)

	wg.Add(1)
	go parseHostForCertFiles(rootPath, paths, errs, &wg)
	go certWorker(paths, errs, certs, &wg)
	go func() {
		wg.Wait()
		done <- true
	}()

	certificates := make(certificates, 0)

	for {
		select {
		case <-done:
			mutex.Lock()
			fmt.Println("Done in:", time.Since(start), "\nParsed", parsedCount, "files\nFound", len(certificates), "out of", validCount, "certificates")
			mutex.Unlock()

			certificateJSON, err := json.Marshal(certificates)
			if err != nil {
				return nil, err
			}
			//fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zw.Name, zw.Comment, zw.ModTime.UTC())
			return certificateJSON, nil
		case cert := <-certs:
			certificates = append(certificates, *cert)
		case err := <-errs:
			fmt.Println("error: ", err)
		}
	}
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

func certWorker(entries chan fsEntry, errs chan error, certs chan *certificate, wg *sync.WaitGroup) {
	for entry := range entries {
		wg.Add(1)
		// too fast, need to save entry value before executing fo routine
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

func parseEntry(entry fsEntry) (*certificate, error) {
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
