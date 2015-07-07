package main

import (
	"os"
	"encoding/json"
	//"fmt"
	"strings"
)

func Registrar(state map[string]*FileState, input chan []*FileEvent) {

	f, err := os.OpenFile("/Users/mac/PycharmProjects/logstash-forwarder/splunk.log", 
					  os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
	    panic(err)
	}

	for events := range input {
		emit ("Registrar: processing %d events\n", len(events))
		// Take the last event found for each file source

		filtered_events := make([]string, len(events))
		counter := 0
		for _, event := range events {
			// skip stdin
			if *event.Source == "-" {
				continue
			}

			ino, dev := file_ids(event.fileinfo)
			state[*event.Source] = &FileState{
				Source: event.Source,
				// take the offset + length of the line + newline char and
				// save it as the new starting offset.
				// This issues a problem, if the EOL is a CRLF! Then on start it read the LF again and generates a event with an empty line
				Offset: event.Offset + int64(len(*event.Text)) + 1, // REVU: this is begging for BUGs
				Inode:  ino,
				Device: dev,
			}

			if strings.Contains(*event.Text, `"msg":"MSG2"`) {
				filtered_events[counter] = *event.Text
				counter++
			}
		}

		for i,element := range filtered_events {
			if i<counter {
  				//fmt.Printf(element)
  				f.WriteString(element + "\n")
  			}
		}

		if e := writeRegistry(state, ".logstash-forwarder"); e != nil {
			// REVU: but we should panic, or something, right?
			emit("WARNING: (continuing) update of registry returned error: %s", e)
		}
	}
}

func writeRegistry(state map[string]*FileState, path string) error {
	tempfile := path + ".new"
	file, e := os.Create(tempfile)
	if e != nil {
		emit("Failed to create tempfile (%s) for writing: %s\n", tempfile, e)
		return e
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(state)

	return onRegistryWrite(path, tempfile)
}
