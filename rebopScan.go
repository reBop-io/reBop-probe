package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

func rebopScan(rootPath string) ([]byte, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		log.Println("Couldn't open %s", rootPath)
		return nil, nil
	}
	//else {
	/*
		var rootPath, err = filepath.Abs(os.Args[1])
		if err != nil {
			fmt.Errorf(err.Error())
			return nil
		}
	*/
	var wg sync.WaitGroup

	paths := make(chan fsEntry, 1)
	errs := make(chan error, 1)
	certs := make(chan *certificate, 1)
	done := make(chan bool, 1)

	wg.Add(1)
	go parseHostForCertFiles(rootPath, paths, errs, &wg)
	go worker(paths, errs, certs, &wg)
	go func() {
		wg.Wait()
		done <- true
	}()

	certificates := make(certificates, 0)

	for {
		select {
		case <-done:
			mutex.Lock()
			fmt.Println("Done in:", time.Since(start), "\nParsed", parsedCount, "files\nSaved", len(certificates), "out of", validCount, "certificates")
			mutex.Unlock()

			certificateJSON, err := json.Marshal(certificates)
			if err != nil {
				fmt.Errorf(err.Error())
				os.Exit(1)
			}
			//fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zw.Name, zw.Comment, zw.ModTime.UTC())
			return certificateJSON, nil
		case cert := <-certs:
			certificates = append(certificates, *cert)
		case err := <-errs:
			fmt.Println("error: ", err)
		}
	}
	//}
}
