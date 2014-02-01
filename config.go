package main

import (
  "encoding/json"
  "errors"
  "log"
  "os"
  "time"
)

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
    config.Network.Timeout = 15
  }

  config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second

  for k, _ := range config.Files {
    if config.Files[k].DeadTime == "" {
      config.Files[k].DeadTime = "24h"
    }
    config.Files[k].deadtime, err = time.ParseDuration(config.Files[k].DeadTime)
    if err != nil {
      log.Printf("Failed to parse dead time duration '%s'. Error was: %s\n", config.Files[k].DeadTime, err)
      return
    }
    // Prospector loops every 10s and due to lack of checks there we can't let dead time be less than this
    // Otherwise the ability to resume on dead files if they return to life will fail
    if config.Files[k].deadtime < 30 * time.Second {
      err = errors.New("Dead time cannot be less than 30 seconds.")
      log.Printf("Dead time cannot be less than 30 seconds. You specified %s.\n", config.Files[k].DeadTime)
      return
    }
  }

  return
}
