package main

import (
	"encoding/json"
	"log"
	"os"
)

func WriteRegistry(state map[string]*FileState, path string) {
	// load data from previous saved file
	historical_state := make(map[string]*FileState)
	history, err := os.Open(path)
	if err == nil {
		decoder := json.NewDecoder(history)
		decoder.Decode(&historical_state)
		history.Close()
	}
	tmp := path + ".new"
	file, err := os.Create(tmp)
	if err != nil {
		log.Printf("Failed to open %s for writing: %s\n", tmp, err)
		return
	}
	// update status with new events
	for p, state := range state {
		historical_state[p] = state
	}

	encoder := json.NewEncoder(file)
	encoder.Encode(historical_state)
	file.Close()

	old := path + ".old"
	os.Rename(path, old)
	os.Rename(tmp, path)
}
