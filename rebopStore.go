package main

import (
	"compress/gzip"
	"log"
	"os"
	"time"
)

func rebopStore(certificateJSON []byte, outfile string) error {
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
	_, err := zw.Write([]byte(certificateJSON))
	if err != nil {
		//log.Println(err)
		log.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		//log.Println(err)
		log.Fatal(err)
		return err
	}
	return nil
}
