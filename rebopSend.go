package main

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/rebop-io/reBop-probe/log"
)

func rebopSend(certArray []byte, filename string, config Config) error {
	var requestBody bytes.Buffer
	multiPartWriter := multipart.NewWriter(&requestBody)
	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Disposition", "form-data; name=\"rebopFile\"; filename=\""+filename+"\"")
	partHeaders.Set("Content-Type", "application/json")
	//partHeaders.Set("Content-Type", "application/gzip")
	fileWriter, err := multiPartWriter.CreatePart(partHeaders)
	if err != nil {
		return err
	}

	/*
		partHeaders.Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(fileWriter)
		_, err = gz.Write(certArray)
		gz.Close()
	*/
	/*
		if compressed {

		}
		else {
	*/
	_, err = fileWriter.Write(certArray)
	//	}
	if err != nil {
		return err
	}

	/*
		_, err = io.Copy(fileWriter, reader)
		if err != nil {
			return err
		}
	*/
	multiPartWriter.Close()

	req, err := http.NewRequest("POST", config.Rebopserver.Proto+"://"+config.Rebopserver.Host+":"+config.Rebopserver.Port+"/files/upload/", &requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Api-Key "+config.User.ReBopAPIKey)
	//fmt.Print(req)
	log.Infof("Connecting to %s with API-Key", req.Host)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	log.Infof("%s responded with %d status code", req.Host, response.StatusCode)
	if response.StatusCode != 200 {
		return errors.New("Can't connect to rebop Server: " + response.Status)
	}
	return nil
}
