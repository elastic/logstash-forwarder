package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"time"
)

const configFileSizeLimit = 10 << 20

var defaultConfig = &struct {
	netTimeout   int64
	fileDeadtime string
}{
	netTimeout:   15,
	fileDeadtime: "24h",
}

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
	Paths    []string          `json:paths`
	Fields   map[string]string `json:fields`
	DeadTime string            `json:"dead time"`
	deadtime time.Duration
}

func DiscoverConfigs(file_or_directory string) (files []string, err error) {
	fi, err := os.Stat(file_or_directory)
	if err != nil {
		return nil, err
	}
	files = make([]string, 0)
	if fi.IsDir() {
		entries, err := ioutil.ReadDir(file_or_directory)
		if err != nil {
			return nil, err
		}
		for _, filename := range entries {
			files = append(files, path.Join(file_or_directory, filename.Name()))
		}
	} else {
		files = append(files, file_or_directory)
	}
	return files, nil
}

// Append values to the 'to' config from the 'from' config, erroring
// if a value would be overwritten by the merge.
func MergeConfig(to *Config, from Config) (err error) {

	to.Network.Servers = append(to.Network.Servers, from.Network.Servers...)
	to.Files = append(to.Files, from.Files...)

	// TODO: Is there a better way to do this in Go?
	if from.Network.SSLCertificate != "" {
		if to.Network.SSLCertificate != "" {
			return fmt.Errorf("SSLCertificate already defined as '%s' in previous config file", to.Network.SSLCertificate)
		}
		to.Network.SSLCertificate = from.Network.SSLCertificate
	}
	if from.Network.SSLKey != "" {
		if to.Network.SSLKey != "" {
			return fmt.Errorf("SSLKey already defined as '%s' in previous config file", to.Network.SSLKey)
		}
		to.Network.SSLKey = from.Network.SSLKey
	}
	if from.Network.SSLCA != "" {
		if to.Network.SSLCA != "" {
			return fmt.Errorf("SSLCA already defined as '%s' in previous config file", to.Network.SSLCA)
		}
		to.Network.SSLCA = from.Network.SSLCA
	}
	if from.Network.Timeout != 0 {
		if to.Network.Timeout != 0 {
			return fmt.Errorf("Timeout already defined as '%d' in previous config file", to.Network.Timeout)
		}
		to.Network.Timeout = from.Network.Timeout
	}
	return nil
}

func LoadConfig(path string) (config Config, err error) {
	config_file, err := os.Open(path)
	if err != nil {
		emit("Failed to open config file '%s': %s\n", path, err)
		return
	}

	fi, _ := config_file.Stat()
	if size := fi.Size(); size > (configFileSizeLimit) {
		emit("config file (%q) size exceeds reasonable limit (%d) - aborting", path, size)
		return // REVU: shouldn't this return an error, then?
	}

	if fi.Size() == 0 {
		emit("config file (%q) is empty, skipping", path)
		return
	}

	buffer := make([]byte, fi.Size())
	_, err = config_file.Read(buffer)
	emit("%s\n", buffer)

	buffer, err = StripComments(buffer)
	if err != nil {
		emit("Failed to strip comments from json: %s\n", err)
		return
	}

	buffer = []byte(os.ExpandEnv(string(buffer)))

	err = json.Unmarshal(buffer, &config)
	if err != nil {
		emit("Failed unmarshalling json: %s\n", err)
		return
	}

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

func FinalizeConfig(config *Config) {
	if config.Network.Timeout == 0 {
		config.Network.Timeout = defaultConfig.netTimeout
	}

	config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second
}

func StripComments(data []byte) ([]byte, error) {
	data = bytes.Replace(data, []byte("\r"), []byte(""), 0) // Windows
	lines := bytes.Split(data, []byte("\n"))
	filtered := make([][]byte, 0)

	for _, line := range lines {
		match, err := regexp.Match(`^\s*#`, line)
		if err != nil {
			return nil, err
		}
		if !match {
			filtered = append(filtered, line)
		}
	}

	return bytes.Join(filtered, []byte("\n")), nil
}
