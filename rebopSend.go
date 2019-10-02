package main

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

func rebopSend(certArray []byte, filename string) {
	reader := bytes.NewReader(certArray)
	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	fileWriter, err := multiPartWriter.CreateFormFile("rebopFile", filename)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		log.Fatalln(err)
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
	req, err := http.NewRequest("POST", "http://localhost:3000/files/upload/", &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Api-Key KQD167R-EFJM6YN-PQZKW1N-4G4S23K")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	//var result map[string]interface{}

	//json.NewDecoder(response.Status).Decode(&result)

	//log.Println(result)
	log.Println(response.Status)
}
