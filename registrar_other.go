// +build !windows

package main

import (
	"encoding/json"
	"log"
	"os"
)

func WriteRegistry(state map[string]*FileState, path string) {
	// load data from previous saved file
	historical_state := make(map[string]*FileState)
	history, err := os.Open(".logstash-forwarder")
	if err == nil {
		log.Printf("Loading old registrar data\n")
		decoder := json.NewDecoder(history)
		decoder.Decode(&historical_state)
		history.Close()
	}
	// Open tmp file, write, flush, rename
	file, err := os.Create(".logstash-forwarder.new")
	if err != nil {
		log.Printf("Failed to open .logstash-forwarder.new for writing: %s\n", err)
		return
	}
	defer file.Close()
	// update status with new events
	for p, state := range state {
		historical_state[p] = state
	}
	encoder := json.NewEncoder(file)
	encoder.Encode(historical_state)

	os.Rename(".logstash-forwarder.new", path)
}
