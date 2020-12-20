package main

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

func rebopSend(certArray []byte, filename string, config Config) error {
	//reader := bytes.NewReader(certArray)
	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	//fileWriter, err := multiPartWriter.CreateFormFile("rebopFile", filename+".json")

	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Disposition", "form-data; name=\"rebopFile\"; filename=\""+filename+".json\"")
	//partHeaders.Set("name", "rebopFile")
	//partHeaders.Set("rebopFile", filename+".json")
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
	req.Header.Set("Authorization", "Api-Key "+config.User.reBopAPIKey)
	fmt.Print(req)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("Can't connect to rebop Server: " + response.Status)
	}
	return nil
}
