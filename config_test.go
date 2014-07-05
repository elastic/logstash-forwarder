package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

// -------------------------------------------------------------------
// test
// -------------------------------------------------------------------

// TestLoadConfig data
// Note var 'expected' must be changed in tandem with this sample conf json.
var configJson = `
{
  "network": {
    "servers": [ "localhost:5043" ],
    "ssl certificate": "./lumberjack.crt",
    "ssl key": "./lumberjack.key",
    "ssl ca": "./lumberjack_ca.crt"
  },
  "files": [
    {
      "paths": [
        "/var/log/*.log",
        "/var/log/messages"
      ],
      "fields": { "type": "syslog" }
    }, {
      "paths": [ "/var/log/apache2/access.log" ],
      "fields": { "type": "apache" }
    }
  ]
}
`

var expected = struct {
	network NetworkConfig
	files   []FileConfig
}{
	network: NetworkConfig{
		Servers:        []string{"localhost:5043"},
		SSLCertificate: "./lumberjack.crt",
		SSLKey:         "./lumberjack.key",
		SSLCA:          "./lumberjack_ca.crt",
	},
	files: []FileConfig{
		FileConfig{
			Paths: []string{
				"/var/log/*.log",
				"/var/log/messages",
			},
			Fields: map[string]string{
				"type": "syslog",
			},
		},
		FileConfig{
			Paths: []string{
				"/var/log/apache2/access.log",
			},
			Fields: map[string]string{
				"type": "apache",
			},
		},
	},
}

// tests main.LoadConfig
func TestLoadConfig(t *testing.T) {
	// set it up
	fname := writeConfFile([]byte(configJson))
	_, e := os.Stat(fname)
	if e != nil {
		testBug(e)
	}

	config, e := LoadConfig(fname)
	if e != nil {
		t.Errorf("filename:%s - error: %s", fname, e)
	}

	// check network
	for i := 0; i < len(expected.network.Servers); i++ {
		if config.Network.Servers[i] != expected.network.Servers[i] {
			t.Errorf("networks do not match: i:%d", i)
		}
	}
	if config.Network.SSLCertificate != expected.network.SSLCertificate {
		t.Errorf("SSLCertificate do not match")
	}
	if config.Network.SSLKey != expected.network.SSLKey {
		t.Errorf("SSLKey do not match")
	}
	if config.Network.SSLCA != expected.network.SSLCA {
		t.Errorf("SSLCA do not match")
	}
	// check files
	for i := 0; i < len(expected.files); i++ {
		for k, v := range expected.files[i].Fields {
			if config.Files[i].Fields[k] != v {
				t.Errorf("fields do not match: i:%d k:%s", i, k)
			}
		}
		for j := 0; j < len(expected.files[i].Paths); j++ {
			if config.Files[i].Paths[j] != expected.files[i].Paths[j] {
				t.Errorf("paths do not match: i:%d j:%d", i, j)
			}
		}
	}
}

// -------------------------------------------------------------------
// test support funcs
// -------------------------------------------------------------------
func writeConfFile(data []byte) string {
	fname := path.Join(os.TempDir(), "test-logstash-forwarder.conf")
	e := ioutil.WriteFile(fname, data, os.FileMode(0644))
	if e != nil {
		panic("TESTBUG: " + e.Error())
	}
	return fname
}

// call on test setup failure.
func testBug(e error) { panic("TESTBUG: " + e.Error()) }
