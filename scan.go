package rebopagent

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

func rebopScan(rootPath string, outfile string) error {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		log.Println("Couldn't open %s", rootPath)
		return nil
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

			var filename = outfile + "-rebop-" + time.Now().Local().Format("2006-01-02") + ".gz"
			// Open the gzip file.
			f, _ := os.Create(filename)
			//var buf bytes.Buffer
			//zw := gzip.NewWriter(&buf)
			zw := gzip.NewWriter(f)

			// Setting the Header fields is optional.
			//zw.Name = filename
			//zw.Comment = "Rebop file"
			//zw.ModTime = time.Date(1977, time.May, 25, 0, 0, 0, 0, time.UTC)

			//test, err := zw.Write([]byte(certificateJSON))
			_, err = zw.Write([]byte(certificateJSON))
			if err != nil {
				//log.Println(err)
				log.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				//log.Println(err)
				log.Fatal(err)
			}

			//fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zw.Name, zw.Comment, zw.ModTime.UTC())

			/*
				err = ioutil.WriteFile(
					outfile+"-rebop-"+time.Now().Local().Format("2006-01-02")+".json",
					//time.Now().Local().Format("2006-01-02")+"-"+hostname+"-rebop.json",
					certificateJSON,
					0644)
				if err != nil {
					fmt.Errorf(err.Error())
					os.Exit(1)
				}
			*/
			return nil
		case cert := <-certs:
			certificates = append(certificates, *cert)
		case err := <-errs:
			fmt.Println("error: ", err)
		}
	}
	//}
}
