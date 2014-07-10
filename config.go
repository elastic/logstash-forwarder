package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

const default_NetworkConfig_Timeout int64 = 15

const default_FileConfig_DeadTime string = "24h"

type Config struct {
	Network NetworkConfig `json:network`
	Files   []FileConfig  `json:files`
}

type NetworkConfig struct {
	Servers        []string `json:servers`
	SSLCertificate string   `json:"ssl certificate"`
	SSLKey         string   `json:"ssl key"`
	SSLCA          string   `json:"ssl ca"`
	Timeout        int64    `json:timeout`
	timeout        time.Duration
}

type FileConfig struct {
  Paths  []string          `json:paths`
  Fields map[string]string `json:fields`
  DeadTime string `json:"dead time"`
  deadtime time.Duration
}

func LoadConfig(path string) (config Config, err error) {
	config_file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open config file '%s': %s\n", path, err)
		return
	}

	fi, _ := config_file.Stat()
	if fi.Size() > (10 << 20) {
		log.Printf("Config file too large? Aborting, just in case. '%s' is %d bytes\n",
			path, fi)
		return
	}

	buffer := make([]byte, fi.Size())
	_, err = config_file.Read(buffer)
	log.Printf("%s\n", buffer)

	err = json.Unmarshal(buffer, &config)
	if err != nil {
		log.Printf("Failed unmarshalling json: %s\n", err)
		return
	}

  if config.Network.Timeout == 0 {
    config.Network.Timeout = default_NetworkConfig_Timeout
  }

	config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second

  for k, _ := range config.Files {
    if config.Files[k].DeadTime == "" {
      config.Files[k].DeadTime = default_FileConfig_DeadTime
    }
    config.Files[k].deadtime, err = time.ParseDuration(config.Files[k].DeadTime)
    if err != nil {
      log.Printf("Failed to parse dead time duration '%s'. Error was: %s\n", config.Files[k].DeadTime, err)
      return
    }
  }

	return
}
