package main

import (
	"expvar"
	"flag"
	"log"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var path string

func init() {
	log.SetFlags(0)
	const (
		usage = "path to track"
	)

	flag.StringVar(&path, "p", ".", usage)
}
var publish chan interface{}
func _var(f func() interface{}) expvar.Func { return expvar.Func(f) }
func main() {
	flag.Parse()

	log.Printf("trak %q", path)
	ctl, requests, reports := tracker()

	expvar.Publish("ctl", _var(func() interface{} { return ctl }))
	expvar.Publish("reports", _var(func() interface{} { return reports }))
	expvar.Publish("requests", _var(func() interface{} { return requests }))
	expvar.Publish("time", _var(func() interface{} { return time.Now() }))
	publish = make(chan interface{}, 1)
	var prev []string = []string{}
	expvar.Publish("hideme", _var(func() interface{} {
		select {
		case report := <-publish:
			prev = append(prev, fmt.Sprintf("%s", report))
			return report
		default:
			return prev
		}
	}))
	user := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)

	go track(*ctl, requests, reports, path, "*")

	go http.ListenAndServe(":12345", nil)

	flag := true
	requests <- struct{}{}
	for flag {
		//		time.Sleep(time.Microsecond)
		select {
		case report := <-reports:
			var pubmap map[string]string
			if len(report.files) > 0 {
				pubmap = make(map[string]string)
				pubmap["timestamp"] = report.timestamp.String()
			}
			for name, dir := range report.files {
				pubmap[name] = fmt.Sprintf("%d %s", dir.Size(), dir.ModTime().String())
				log.Printf("%s %s %d ", name, dir.Name(), dir.Size())
			}
			log.Println()
			if len(report.files) > 0 {
				publish <- pubmap
			}
			time.Sleep(time.Millisecond * 100)
			requests <- struct{}{}
		case stat := <-ctl.stat:
			log.Println("stat: %s", stat)
			flag = false
		case <-user:
			os.Exit(0)
		default:
		}
	}
	stat := <-ctl.stat
	log.Printf("stat: %s", stat)
}

func tracker() (*control, chan struct{}, chan *trackreport) {
	r := make(chan struct{}, 1)
	c := make(chan *trackreport, 0)
	return procControl(), r, c
}

func procControl() *control {
	return &control{
		sig:  make(chan interface{}, 1),
		stat: make(chan interface{}, 1),
	}
}

type control struct {
	sig  chan interface{}
	stat chan interface{}
}

func track(ctl control, requests <-chan struct{}, out chan<- *trackreport, path string, pattern string) {
	defer recovery(ctl, "done")

	log.Println("traking..")

	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)

	for {
		select {
		case <-requests:
			file, e := os.Open(path)
			anomaly(e)
			dirs, e := file.Readdirnames(0)
			anomaly(e)
			if len(dirs) == 0 {
				panic("zero")
			}
			e = file.Close()
			anomaly(e)

			files := make(map[string]os.FileInfo)
			files0 := make(map[string]os.FileInfo)
			for _, dir := range dirs {
				info, e := os.Stat(dir)
				anomaly(e)
				if dir[0] == '.' {
					continue
				}
				files0[dir] = info
				info0 := snapshot[dir]
				news := false
				// is it news?
				if info0 != nil {
					// compare
					if info0.Size() != info.Size() {
						// changed
						log.Printf("changed %s", dir)
						publish<- fmt.Sprintf("%s changed", dir)
						news = true
					}
				} else {
					log.Printf("initial %s", dir)
					publish<- fmt.Sprintf("%s initial", dir)
					news = true
				}
				if news {
					files[dir] = info
					snapshot[dir] = info
				}
			}
			for f, _ := range snapshot {
				if _, found := files0[f]; !found {
					log.Printf("deleted %s", f)
					publish<- fmt.Sprintf("%s deleted", f)
					delete(snapshot, f)
				}
			}
			//			if len(files) > 0 {
			report := trackreport{files, time.Now()}
			out <- &report
			//			}
		}
	}
}

type trackreport struct {
	files     map[string]os.FileInfo `json: "Files"`
	timestamp time.Time
}

func recovery(ctl control, ok interface{}) {
	log.Println("recovery ..")
	p := recover()

	if p != nil {
		ctl.stat <- p
		return
	}
	ctl.stat <- ok
}
func anomaly(e error) {
	if e != nil {
		panic(e)
	}
}
