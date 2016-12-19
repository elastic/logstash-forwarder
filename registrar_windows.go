package main

import (
	"os"
)

func onRegistryWrite(path, tempfile string) error {
	old := path + ".old"
	var e error

	os.Remove(old);
	if e = os.Rename(path, old); e != nil {
		emit("registry rotate: rename of %s to %s - %s\n", path, old, e)
		return e
	}

	os.Remove(path);
	if e = os.Rename(tempfile, path); e != nil {
		emit("registry rotate: rename of %s to %s - %s\n", tempfile, path, e)
		return e
	}
	return nil
}
