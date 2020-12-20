package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"time"
)

func rebopStore(certificateJSON []byte, outfile string) (string, error) {
	var filename = outfile + "_rebop_" + time.Now().Local().Format("2006-01-02") + ".gz"
	// Open the gzip file.
	f, _ := os.Create(filename)
	zw := gzip.NewWriter(f)

	_, err := zw.Write([]byte(certificateJSON))
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	if err := zw.Close(); err != nil {
		fmt.Println(err)
		return "", err
	}
	return filename, nil
}
