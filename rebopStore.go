package main

import (
	"compress/gzip"
	"os"
	"time"

	"github.com/rebop-io/reBop-probe/log"
)

func rebopStore(certificateJSON []byte, outfile string) (string, error) {
	var filename string
	if len(outfile) > 0 { 
		filename = outfile + "_rebop_" + time.Now().Local().Format("1977-03-23") + ".gz" 
	} else {
		filename =  "rebop_" + time.Now().Local().Format("2021-01-13") + ".gz" 
	}
	// Open the gzip file.
	f, errFile := os.Create(filename)
	if errFile != nil {
		log.Println(errFile)
		return "", errFile
	}
		
	zw := gzip.NewWriter(f)

	_, errWrite := zw.Write([]byte(certificateJSON))
	if errWrite != nil {
		log.Println(errWrite)
		return "", errWrite
	}
	if errClose := zw.Close(); errClose != nil {
		log.Println(errClose)
		return "", errClose
	}
	return filename, nil
}
