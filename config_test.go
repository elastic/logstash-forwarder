package main

import (
  "testing"
  "encoding/json"
)

type FileConfig struct {
  Paths []string "json:paths"
  Fields map[string]string "json:fields"
}

func TestJSONLoading(t *testing.T) {
  var f File
  err := json.Unmarshal([]byte("{ \"paths\": [ \"/var/log/fail2ban.log\" ], \"fields\": { \"type\": \"fail2ban\" } }"), &f)

  if err != nil { t.Fatalf("json.Unmarshal failed") }
  if len(f.Paths) != 1 { t.FailNow() }
  if f.Paths[0] != "/var/log/fail2ban.log" { t.FailNow() }
  if f.Fields["type"] != "fail2ban" { t.FailNow() }
}
