package main

import (
	"os"
	"strings"
	"io/ioutil"
	"encoding/json"
	"time"
)

const CONFIG_FILE string = "./filter.conf"

type ConfigType struct {
	Filters FilterType `json:filters`
}
 
type FilterType struct {
    Exprs  []string `json:exprs`
	Output string   `json:output`
}   

func readConfigFile (config *ConfigType) error {
	file, e := ioutil.ReadFile(CONFIG_FILE)
    if e != nil {
        emit ("Filter: not found ./filter.conf file.\n")
        emit ("Filter: not filtering!.\n")
        return e
    }

    e = json.Unmarshal(file, config)
    if e != nil {
    	emit("Filter: error parsing filter config file.\n")
    	os.Exit(1)
    }

    return e
}

func Filter(input chan []*FileEvent) {
	var config ConfigType
	e := readConfigFile(&config)

	if e != nil {
		return
	}

    lastConfigReload := time.Now().Unix()

	f, err := os.OpenFile(config.Filters.Output, 
					      os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
	    emit("Filter: wrong output file.\n")
    	os.Exit(1)
	}

	for {
		for events := range input {
			info, err := os.Stat(CONFIG_FILE)

			if err == nil {
				if info.ModTime().Unix() > lastConfigReload {
					readConfigFile(&config)
					lastConfigReload = time.Now().Unix()
					emit ("Filter: changed filter expr\n")
				}
			}

			filteredEvents := make([]string, len(events))
			counter := 0
			for _, event := range events {
				// skip stdin
				if *event.Source == "-" {
					continue
				}

				for _, expr := range config.Filters.Exprs {
					if strings.Contains(*event.Text, expr) {
						filteredEvents[counter] = *event.Text
						counter++
						break
					}
				}
			}

			emit ("Filter: filtered %d events\n", counter)

			for i,element := range filteredEvents {
				if i<counter {
	  				//emit("FILTERED TEXT: " + element)
	  				f.WriteString(element + "\n")
	  			}
			}
		}
	}
}
