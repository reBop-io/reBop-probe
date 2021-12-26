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
	f, errFile := os.Create(filename)
	if errFile != nil {
		fmt.Println(errFile)
		return "", errFile
	}
		
	zw := gzip.NewWriter(f)

	_, errWrite := zw.Write([]byte(certificateJSON))
	if errWrite != nil {
		fmt.Println(errWrite)
		return "", errWrite
	}
	if errClose := zw.Close(); errClose != nil {
		fmt.Println(errClose)
		return "", errClose
	}
	return filename, nil
}
