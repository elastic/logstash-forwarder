package main

import (
	"encoding/json"
	"os"
	"path"
	"io/ioutil"
	"time"
)

const configFileSizeLimit = 10 << 20

var defaultConfig = &struct {
		netTimeout int64
		fileDeadtime string
	}{
	netTimeout: 15,
	fileDeadtime: "24h",
}

type Config struct {
	Network          NetworkConfig `json:network`
	Files            []FileConfig  `json:files`
	IncludeFilesDirs []string      `json:includeFilesDirs`
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
	Paths    []string          `json:paths`
	Fields   map[string]string `json:fields`
	DeadTime string            `json:"dead time"`
	deadtime time.Duration
}

func LoadConfig(configPath string) (config Config, err error) {
	config_file, err := os.Open(configPath)
	if err != nil {
		emit("Failed to open config file '%s': %s\n", configPath, err)
		return
	}

	fi, _ := config_file.Stat()
	if size := fi.Size(); size > (configFileSizeLimit) {
		emit("config file (%q) size exceeds reasonable limit (%d) - aborting", configPath, size)
		return // REVU: shouldn't this return an error, then?
	}

	buffer := make([]byte, fi.Size())
	_, err = config_file.Read(buffer)
	emit("%s\n", buffer)

	err = json.Unmarshal(buffer, &config)
	if err != nil {
		emit("Failed unmarshalling json: %s\n", err)
		return
	}
	
	for _, includeDir := range config.IncludeFilesDirs {
		var files []os.FileInfo
		
		files, err = ioutil.ReadDir(includeDir)
		if err != nil {
			emit("Failed reading directory %s: %s\n", includeDir, err)
			return
		}
		
		for _, f := range files {
			fqPath := path.Join(includeDir, f.Name())

			var buf []byte
			
			buf, err = ioutil.ReadFile(fqPath)
			if err != nil {
				emit("Failed reading %s: %s\n", fqPath, err)
				return
			}
			
			fileConfig := FileConfig{}
			err = json.Unmarshal(buf, &fileConfig)
			if err != nil {
				emit("Failed unmarshaling %s: %s\n", fqPath, err)
				return
			}
			
			config.Files = append(config.Files, fileConfig)
		}
	}

	if config.Network.Timeout == 0 {
		config.Network.Timeout = defaultConfig.netTimeout
	}

	config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second

	for k, _ := range config.Files {
		if config.Files[k].DeadTime == "" {
			config.Files[k].DeadTime = defaultConfig.fileDeadtime
		}
		config.Files[k].deadtime, err = time.ParseDuration(config.Files[k].DeadTime)
		if err != nil {
			emit("Failed to parse dead time duration '%s'. Error was: %s\n", config.Files[k].DeadTime, err)
			return
		}
	}

	return
}
