package main

import (
	"io/ioutil"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func setupRegistrarTests(t *testing.T) (string, os.FileInfo) {
	f, err := ioutil.TempFile("", "lsf-registrar")
	if err != nil {
		t.Error("Failed to create file", err)
	}

	stat, err := f.Stat()
	if err != nil {
		t.Error("Failed to stat file", err)
	}

	return f.Name(), stat
}

func teardownRegistrarTests(f string) {
	os.Remove(options.registryFile)
	os.Remove(f)
}

func TestRegistrar(t *testing.T) {
	file, stat := setupRegistrarTests(t)
	defer teardownRegistrarTests(file)

	testState := make(map[string]*FileState)
	testInput := make(chan []*FileEvent)

	mockEvents := []*FileEvent {
		&FileEvent {
			Source: &file,
			Offset: 1024,
			Text: &file,
			fileinfo: &stat,
		},
	}

	go func() {
		testInput <- mockEvents
		close(testInput)
	}()

	Registrar(testState, testInput)

	rf, err := os.Open(options.registryFile)
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

