package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"
)

// -------------------------------------------------------------------
// test support funcs
// -------------------------------------------------------------------
func chkerr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Error encountered: %s", err)
	}
}

func makeTempDir(t *testing.T) string {
	tmpdir, err := ioutil.TempDir("", "logstash-config-test")
	chkerr(t, err)
	return tmpdir
}

func rmTempDir(tmpdir string) {
	_ = os.RemoveAll(tmpdir)
}

// -------------------------------------------------------------------
// Tests
// -------------------------------------------------------------------
func TestDiscoverConfigs(t *testing.T) {
	tmpdir := makeTempDir(t)
	defer rmTempDir(tmpdir)
	tmpfile1 := path.Join(tmpdir, "myfile1")
	tmpfile2 := path.Join(tmpdir, "myfile2")
	err := ioutil.WriteFile(tmpfile1, make([]byte, 0), 0644)
	chkerr(t, err)
	err = ioutil.WriteFile(tmpfile2, make([]byte, 0), 0644)

	configs, err := DiscoverConfigs(tmpdir)
	chkerr(t, err)

	expected := []string{tmpfile1, tmpfile2}
	if !reflect.DeepEqual(configs, expected) {
		t.Fatalf("Expected to find %v, got %v instead", configs, expected)
	}

	configs, err = DiscoverConfigs(tmpfile1)

	expected = []string{tmpfile1}
	if !reflect.DeepEqual(configs, expected) {
		t.Fatalf("Expected to find %v, got %v instead", configs, expected)
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	tmpdir := makeTempDir(t)
	defer rmTempDir(tmpdir)

	configFile := path.Join(tmpdir, "myconfig")
	err := ioutil.WriteFile(configFile, []byte(""), 0644)
	chkerr(t, err)

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Error loading config file: %s", err)
	}

	if !reflect.DeepEqual(config, Config{}) {
		t.Fatalf("Expected emtpy Config, got \n\n%v\n\n from LoadConfig", config)
	}
}

func TestLoadConfigAndStripComments(t *testing.T) {
	configJson := `
# A comment at the beginning of the line
{
  # A comment after some spaces
  "network": {
    "servers": [ "localhost:5043" ],
    "ssl certificate": "./logstash-forwarder.crt",
    "ssl key": "./logstash-forwarder.key",
    "ssl ca": "./logstash-forwarder.ca",
    "timeout": 20
  },
  # A comment in the middle of the JSON
  "files": [
    {
      "paths": [
        "/var/log/*.log",
        "/var/log/messages"
      ],
      "fields": { "type": "syslog" },
      "dead time": "6h"
    }, {
      "paths": [ "/var/log/apache2/access.log" ],
      "fields": { "type": "apache" }
    }
  ]
}`

	tmpdir := makeTempDir(t)
	defer rmTempDir(tmpdir)

	configFile := path.Join(tmpdir, "myconfig")
	err := ioutil.WriteFile(configFile, []byte(configJson), 0644)
	chkerr(t, err)

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Error loading config file: %s", err)
	}

	defaultDeadTime, _ := time.ParseDuration(defaultConfig.fileDeadtime)
	expected := Config{
		Network: NetworkConfig{
			Servers:        []string{"localhost:5043"},
			SSLCertificate: "./logstash-forwarder.crt",
			SSLKey:         "./logstash-forwarder.key",
			SSLCA:          "./logstash-forwarder.ca",
			Timeout:        20,
		},
		Files: []FileConfig{{
			Paths:    []string{"/var/log/*.log", "/var/log/messages"},
			Fields:   map[string]string{"type": "syslog"},
			DeadTime: "6h",
			deadtime: 21600000000000,
		}, {
			Paths:    []string{"/var/log/apache2/access.log"},
			Fields:   map[string]string{"type": "apache"},
			DeadTime: defaultConfig.fileDeadtime,
			deadtime: defaultDeadTime,
		}},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("Expected\n%v\n\ngot\n\n%v\n\nfrom LoadConfig", expected, config)
	}

}

func TestFinalizeConfig(t *testing.T) {
	tests := []struct {
		config   Config
		validate func(c Config) error
	}{
		{
			// Uses correct default timeout when no explicit timeout is set
			config: Config{},
			validate: func(c Config) error {
				if c.Network.Timeout != defaultConfig.netTimeout {
					return fmt.Errorf("Expected FinalizeConfig to default timeout to %d, got %d instead", defaultConfig.netTimeout, c.Network.Timeout)
				}
				return nil
			},
		},
		{
			// When timeout is explicitly set it's converted to time.Duration
			config: Config{
				Network: NetworkConfig{
					Timeout: 40,
				},
			},
			validate: func(c Config) error {
				expected := time.Duration(40) * time.Second
				if c.Network.timeout != expected {
					return fmt.Errorf("Expected FinalizeConfig to set the timeout duration to %v, got %v instead", c.Network.timeout, expected)
				}
				return nil
			},
		},
	}

	for testidx, test := range tests {
		FinalizeConfig(&test.config)
		if test.validate != nil {
			err := test.validate(test.config)
			if err != nil {
				t.Errorf("Test %d: %s", testidx, err)
			}
		}
	}
}

func TestMergeConfig(t *testing.T) {
	configA := Config{
		Network: NetworkConfig{
			Servers:        []string{"localhost:5043"},
			SSLCertificate: "./logstash-forwarder.crt",
			SSLKey:         "./logstash-forwarder.key",
		},
		Files: []FileConfig{{
			Paths: []string{"/var/log/messagesA"},
		}},
	}

	configB := Config{
		Network: NetworkConfig{
			Servers: []string{"otherhost:5043"},
			SSLCA:   "./logstash-forwarder.crt",
			Timeout: 20,
		},
		Files: []FileConfig{{
			Paths: []string{"/var/log/messagesB"},
		}},
	}

	expected := Config{
		Network: NetworkConfig{
			Servers:        []string{"localhost:5043", "otherhost:5043"},
			SSLCertificate: "./logstash-forwarder.crt",
			SSLKey:         "./logstash-forwarder.key",
			SSLCA:          "./logstash-forwarder.crt",
			Timeout:        20,
		},
		Files: []FileConfig{{
			Paths: []string{"/var/log/messagesA"},
		}, {
			Paths: []string{"/var/log/messagesB"},
		}},
	}

	err := MergeConfig(&configA, configB)
	chkerr(t, err)

	if !reflect.DeepEqual(configA, expected) {
		t.Fatalf("Expected merged config to be %v, got %v instead", expected, configA)
	}

	err = MergeConfig(&configA, configB)
	if err == nil {
		t.Fatalf("Expected a double merge attempt to give us an error, it didn't")
	}
}
