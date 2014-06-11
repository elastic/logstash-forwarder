package lsf

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// TODO: get rid of this
type CancelChan <-chan struct{}

type StringFn interface {
	String() string
}

// convenience function Pretty print some common (std lib) type values
// that do not implement the String() interface.
func Sprint(v interface{}) string {
	switch t := v.(type) {
	case os.FileInfo:
		return fmt.Sprintf("fileInfo name:%s size:%d mode:%d mod-time:%d", t.Name(), t.Size(), t.Mode(), t.ModTime().UnixNano())
	}
	return fmt.Sprintf("%v", v)
}

// Convenience function Joins the strings using a space.
func Str(s ...string) string {
	return strings.Join(s, " ")
}

type resetFn func()

// panics on os.Getwd (as that should be error free)
func cd(path string) (resetFn, error) {
	wd, e := os.Getwd()
	if e != nil {
		return nil, e
	}

	e = os.Chdir(path)
	if e != nil {
		return nil, e // errorWithCause + error codes for commands ..
	}

	return func() {
		e = os.Chdir(wd)
		if e != nil {
			panic("fault on reset os.Chdir(" + wd + ")")
		}
	}, nil
}

func Wd() string {
	wd, _ := os.Getwd()
	return wd
}

// Return absolute path per working directory
func AbsolutePath(p string) (abspath string) {
	if path.IsAbs(p) {
		abspath = p
	} else {
		abspath = path.Join(Wd(), p)
	}
	return
}
func IsAbsPath(p string) bool {
	return path.IsAbs(p)
}

// returns (lower case) hex representation of
// of the SHA1 hash of the string s
func HexShaDigest(s string) string {
	md := sha1.New()
	io.WriteString(md, s)
	return fmt.Sprintf("%x", md.Sum(nil))
}
