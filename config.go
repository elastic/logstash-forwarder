package main

import (
  "encoding/json"
  "os"
  "log"
  "time"
)

type Config struct {
  Lumberjack AppConfig `json:lumberjack`
  Network NetworkConfig `json:network`
  Files []FileConfig `json:files`
}

type AppConfig struct {
  RegistrarFile string `json:"registrar file"`
  IgnoreAfter int64 `json:"ignore after"`
  DeadTime  int64 `json:dead time`
}

type NetworkConfig struct {
  Servers []string `json:servers`
  SSLCertificate string `json:"ssl certificate"`
  SSLKey string `json:"ssl key"`
  SSLCA string `json:"ssl ca"`
  Timeout int64 `json:timeout`
  timeout time.Duration
} 

type FileConfig struct {
  Paths []string `json:paths`
  Fields map[string]string `json:fields`
  //DeadTime time.Duration `json:"dead time"`
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

  //Application Config Defaults
  if config.Lumberjack.RegistrarFile == "" {
    log.Print("Using default registrar file: .lumberjack")
    config.Lumberjack.RegistrarFile = ".lumberjack"
  }
  if config.Lumberjack.IgnoreAfter == 0 {
    config.Lumberjack.IgnoreAfter = 24
  }
  if config.Lumberjack.DeadTime == 0 {
    config.Lumberjack.DeadTime = 24
  }

  //for _, fileconfig := range config.Files {
    //if fileconfig.DeadTime == 0 {
      //fileconfig.DeadTime = 24 * time.Hour
    //}
  //}

  return
}
