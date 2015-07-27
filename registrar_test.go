package main

import (
	"io/ioutil"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func setupRegistrarTests(t *testing.T) (string, string, os.FileInfo) {
	lf, err := ioutil.TempFile("", "lsf-registrar")
	if err != nil {
		t.Error("Failed to create log file", err)
	}

	stat, err := lf.Stat()
	if err != nil {
		t.Error("Failed to stat file", err)
	}

	rf, err := ioutil.TempFile("", "lsf-registry")
	if err != nil {
		t.Error("Failed to create registry file", err)
	}

	return rf.Name(), lf.Name(), stat
}

func teardownRegistrarTests(lf, rf string) {
	os.Remove(lf)
	os.Remove(rf)
}

func TestRegistrar(t *testing.T) {
	registryFile, logFile, stat := setupRegistrarTests(t)
	defer teardownRegistrarTests(registryFile, logFile)

	testState := make(map[string]*FileState)
	testInput := make(chan []*FileEvent)

	mockEvents := []*FileEvent {
		&FileEvent {
			Source: &logFile,
			Offset: 1024,
			Text: &logFile,
			fileinfo: &stat,
		},
	}

	go func() {
		testInput <- mockEvents
		close(testInput)
	}()

	Registrar(testState, testInput, registryFile)

	rf, err := os.Open(registryFile)
	if err != nil {
		t.Fatal("Failed to open registry file", err)
	}

	registry := make(map[string]*FileState)
	decoder := json.NewDecoder(rf)
	decoder.Decode(&registry)

	if ! reflect.DeepEqual(testState, registry) {
		t.Fatalf("\nexpected: %s\ngot: %s\n", testState, registry)
	}
}

