// Licensed to Elasticsearch under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"flag"
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf/lslib"
	"os"
	"time"
)

var sequence int64

func simulateLogInput() []byte {
	line := fmt.Sprintf("%d %019d INFO simulated single line sequenced log entry\n", time.Now().UnixNano(), sequence)
	sequence++
	return []byte(line)
}

var config struct {
	path, filename string
	maxsize        int64
	maxfiles       uint
	delay          time.Duration
	fileperm       os.FileMode
}

// Options -name is required.
var delayOpt uint
var filemode uint

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.path, "path", ".", "path to log file dir")
	flag.StringVar(&config.filename, "name", "", "basename for log files")
	flag.Int64Var(&config.maxsize, "size", int64(16777216), "max size of each log file")
	flag.UintVar(&config.maxfiles, "num", uint(16), "max number of rotated filesa")
	flag.UintVar(&delayOpt, "f", uint(10), "microsec delay between each log event")
	flag.UintVar(&filemode, "m", uint(0644), "microsec delay between each log event")

	panics.DEBUG = true
}

// Simulate a rotating log writer.
// See init() for option details.
func main() {

	flag.Parse()
	config.delay = time.Duration(delayOpt) * time.Microsecond
	config.fileperm = os.FileMode(0644)
	if config.filename == "" {
		log.Println("option -name is required.")
		flag.Usage()
		os.Exit(0)
	}
	stop, wdone := newProcess()
	go writeLog(config.path, config.filename, config.maxsize, config.maxfiles, config.delay, stop, wdone)

	log.Println(<-wdone)
}

func writeLog(basepath, basename string, maxFileSize int64, maxfiles uint, delay_msec time.Duration, stop <-chan interface{}, wdone chan<- interface{}) {
	defer panics.AsyncRecover(wdone, "ok")

	rotator, e := lslib.NewRotatingFileWriter(basepath, basename, maxfiles, maxFileSize)
	panics.OnError(e, "NewFileRotator")

	for {
		select {
		case <-stop:
			log.Printf("writer STOP\n")
			wdone <- true
			close(wdone)
			return
		default:
			_, e := rotator.Write(simulateLogInput())
			panics.OnError(e, "rotator.Write")

			time.Sleep(delay_msec)
		}
	}
}

func newProcess() (stop <-chan interface{}, wdone chan interface{}) {
	return make(chan interface{}, 1), make(chan interface{}, 1)
}
