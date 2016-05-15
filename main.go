package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	port := flag.Int("port", 8080, "port number")
	configFilename := flag.String("conf", "config.json", "configuration file")
	flag.Parse()

	json, err := ioutil.ReadFile(*configFilename)
	if err != nil {
		log.Fatalf("Failed to open configuration file '%s': '%v'", *configFilename, err)
	}

	config, err := LoadJsonConfig(json)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	service := CreateService(config)
	service.Start(fmt.Sprintf("127.0.0.1:%d", *port))
}
