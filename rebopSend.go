package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

func rebopSend(certArray []byte, filename string, config Config) error {
	reader := bytes.NewReader(certArray)
	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	fileWriter, err := multiPartWriter.CreateFormFile("rebopFile", filename+".json")
	if err != nil {
		log.Fatalln(err)
		return err
	}

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	//fieldWriter, err := multiPartWriter.CreateFormField("test")
	//if err != nil {
	//	log.Fatalln(err)
	//}

	//_, err = fieldWriter.Write([]byte("Value"))
	//if err != nil {
	//	log.Fatalln(err)
	//}

	multiPartWriter.Close()
	req, err := http.NewRequest("POST", config.Rebopserver.Proto+"://"+config.Rebopserver.Host+":"+config.Rebopserver.Port+"/files/upload/", &requestBody)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Api-Key "+config.User.Rebopapikey)

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("Can't connect to rebop Server: " + response.Status)
	}
	return nil
}
