package main

import (
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
	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "production"
	}

	dirname, err := os.Executable()
	if err != nil {
		log.Println(err)
	}
	filename := []string{"/config/", "config.", env, ".yml"}
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
