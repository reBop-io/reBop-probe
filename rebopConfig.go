package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// getConfigFileName ... "Get rebop config file name from ENV"
func getConfigFileName() string {
	env := os.Getenv("REBOP_ENV")
	if len(env) != 0 {
		env = "." + env
	} else {
		env = ""
	}

	dirname, err := os.Executable()
	if err != nil {
		log.Println(err)
	}
	filename := []string{"/config/", "config", env, ".yml"}
	//_, dirname, _, _ := runtime.Caller(0)
	filePath := path.Join(filepath.Dir(dirname), strings.Join(filename, ""))

	return filePath
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func getrebopConfig(cfg *Config) {
	f, err := os.Open(getConfigFileName())
	if err != nil {
		processError(err)
	}

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		processError(err)
	}
}

// Local DB

// Load loads the file at path into v.
// Use os.IsNotExist() to see if the returned error is due
// to the file being missing.
func loadLocalDB(path string, v interface{}) error {
	// Open a RO file
	decodeFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer decodeFile.Close()
	// Create a decoder
	decoder := gob.NewDecoder(decodeFile)

	decoder.Decode(v)
	return err
}

// Save saves a representation of v to the file at path.
func saveLocaDB(path string, v interface{}) error {
	// Create a file for IO
	encodeFile, err := os.Create(path)
	if err != nil {
		return err
	}
	// Since this is a binary format large parts of it will be unreadable
	encoder := gob.NewEncoder(encodeFile)
	// Write to the file
	if err := encoder.Encode(v); err != nil {
		return err
	}
	encodeFile.Close()
	return err
}
