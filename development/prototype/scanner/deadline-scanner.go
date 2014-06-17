package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"flag"
)


// ----------------------------------------------------------------------------------------------------------------
/// SIMULATE LSF Stream Command
// ----------------------------------------------------------------------------------------------------------------

var time_never time.Time = time.Time{}

var config struct {
	path  string
}

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.path, "path", "", "per stream path")
}

// NOTE: of course, you are running a (faux) logger, right? :)
func main() {
	log.SetFlags(0)

	stream := "foo.bar"
	reportsChan := make(chan []*TrackReport, 2)
	requestChan := make(chan string, 2)
	cancel := make(chan interface{}, 1)

	go TrackStream("foo.bar", requestChan, reportsChan, cancel)

	requestChan <- stream // i.e. track "foo.bar*" in current dir

waitnext:
	for {
		select {
		case reports := <-reportsChan:
			if len(reports) == 0 {
				//				time.Sleep(time.Second * 1)
				requestChan <- stream
				continue waitnext
			}
			forwardStream(stream, reports, requestChan)
		case <-time.After(time.Minute * 3):
			stop <- true
			break
		}
	}
}

func TrackStream(stream string, requestChan <-chan string, reportsChan chan<- []*TrackReport, cancel <-chan interface{}) {

	for {
		select {
		case request := <-requestChan:
			//			fmt.Println("Tracking: process request", request)
			// TODO: request needs tobe a struct
			pattern := fmt.Sprintf("%s*", request)
			var reports []*TrackReport
			GlobApply(pattern, DoGenerateTrackReport(&reports))
			reportsChan <- reports
		}
	}
}

func forwardStream(stream string, trackReports []*TrackReport, trackingRequests chan<- string) {

	scanReport := LoadScanReportFile("scan.report") // may be nil
	info, offset, e := DetermineNextScanSpecBasedOnTracking(scanReport, trackReports)
	if e != nil {
		fmt.Printf("ERR - %s\n", e)
		time.Sleep(time.Second)
		trackingRequests <- stream
		return
	}
	fmt.Printf("SCAN - %s @ %d\n", (*info).Name(), offset)
	file, e := os.Open((*info).Name())
	if e != nil {
		panic(e)
	}
	defer file.Close()

	// pipeline plumbing
	fscanevents := make(chan interface{}, 2)
	blocksout := make(chan []byte, 2)
	linesout := make(chan []byte, 2)
	bundleout := make(chan *Bundle, 2)

	// pipeline control
	//	stop := make(chan interface{}, 1)
	sdone := make(chan interface{}, 1)
	bdone := make(chan interface{}, 1)

	// PL 1
	//	offset = GetOffsetFromReport()

	deadline := time.Now().Add(time.Second * 10)
	flushwait := time.Millisecond * 100
	blocksize := 512 * 2 * 64 * 1
	go ScanBlocks(fscanevents, file, offset, deadline, blocksize, flushwait) // 644446 events - 2629397753 bytes - 119976137054 nsecs - ev/sec:5415 b/sec:22095779

	go func() {
		var n int64 = 0
		var cnt int64 = 0
		start := time.Now()
		var delta time.Duration
		for event := range fscanevents {
			cnt++
			switch t := event.(type) {
			case *ScanStart:
			case *ScanReport:
				writeScanReport(t) // expecting fscanevents close after this ..
				fmt.Printf("ScanReport: %s\n", t)
				trackingRequests <- stream
			case *FileBlock:
				n += int64(len(t.data))
				blocksout <- t.data
			}
			delta = time.Now().Sub(start)
		}
		deltasec := delta.Nanoseconds() / 1000000000
		epersec := cnt / deltasec
		bpersec := n / deltasec
		log.Printf("%d events - %d bytes - %d nsecs - ev/sec:%d b/sec:%d \n", cnt, n, delta, epersec, bpersec)

		close(blocksout)
		sdone <- true
	}()

	// PL 2
	go ScanLines(blocksout, linesout)

	// PL 3
	go BundleLines(linesout, bundleout, 480, time.Microsecond*100)

	// PL 4 PIPE OUT
	// SINK
	go func() {
		bundles := 0
		lines := 0
		for bundle := range bundleout {
			lines += bundle.len
			bundles++
			//			fmt.Printf("---------- bundle size: %d\n", bundle.len)
			//			for i:=0; i<bundle.len; i++ {
			//				fmt.Println(string(bundle.lines[i]))
			//			}
		}
		fmt.Printf("BUNDLE - %d bundles - %d lines -- END\n", bundles, lines)
		bdone <- true
	}()

	// deadline for file scan will trip this.
	<-sdone
	<-bdone
}

func GetOffsetFromReport() int64 {
	var offset int64 = 0
	scanReport := readScanReportFile("scan.report") // would be normally under folder 'stream-x/' so same name always
	if scanReport != nil {
		//		fmt.Printf("scan report: %s\n", scanReport)
		offset = scanReport.Offsets[1]
	}
	return offset
}

type ScanStart struct {
	file      os.FileInfo
	offset    int64
	timestamp time.Time
}
type FileBlock struct {
	data   []byte
	offset int64
}
type ScanReport struct {
	file       os.FileInfo
	Inode      uint64
	Device     int64
	Offsets    [2]int64
	Timestamps [2]time.Time
}

func (t *ScanReport) String() string {
	return fmt.Sprintf("%d %d %d %d %d %d", t.Inode, t.Device, t.Offsets[0], t.Offsets[1], t.Timestamps[0].UnixNano(), t.Timestamps[1].UnixNano())
}

func WriteScanReport(r *ScanReport) string {
	ino, dev := FileSysId(&r.file)
	s := fmt.Sprintf("%d %d %d %d %d %d", ino, dev, r.Offsets[0], r.Offsets[1], r.Timestamps[0].UnixNano(), r.Timestamps[1].UnixNano())
	fmt.Println("SCAN-REPORT", s)
	//	fmt.Printf("---Write------------------ INODE: %d\n", ino)

	return s
}

// TEMP - used by the channel processor to handle ScanReport events.
// should be UpdateScanReportRecord
func writeScanReport(r *ScanReport) {
	file, e := os.OpenFile("scan.report", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if e != nil {
		panic(e)
	}
	defer file.Close()

	file.Seek(0, os.SEEK_SET)

	file.WriteString(WriteScanReport(r))
	file.WriteString("\n")
}

func ReadScanReport(s string) *ScanReport {
	defer func() {
		p := recover()
		if p != nil {
			panic(p)
		}
	}()
	var offset0, offset1 int64
	var time0, time1 int64
	var inode, device int64
	fmt.Sscanf(s, "%d %d %d %d %d %d\n", &inode, &device, &offset0, &offset1, &time0, &time1)

	return &ScanReport{
		Inode:      uint64(inode),
		Device:     device,
		Offsets:    [2]int64{offset0, offset1},
		Timestamps: [2]time.Time{time.Unix(0, time0), time.Unix(0, time1)},
	}
}
func readScanReportFile(fname string) *ScanReport {

	file, e := os.Open(fname)
	if e != nil {
		return nil
	}
	defer file.Close()

	b := bufio.NewReader(file)
	line, e := b.ReadString('\n')
	if e != nil {
		panic(e)
	}
	scanReport := ReadScanReport(line)
	return scanReport
}

func FileSysId(info *os.FileInfo) (uint64, int32) {
	fstat := (*(info)).Sys().(*syscall.Stat_t)
	//	fmt.Println(fstat)
	return fstat.Ino, fstat.Dev
}

func (t *FileBlock) String() string {
	return fmt.Sprintf("%09d: '%4d'...", t.offset, len(t.data))
}

func processEvent(ev interface{}) int64 {
	switch t := ev.(type) {
	case *ScanStart:
	case *ScanReport:
		writeScanReport(t)
	case *FileBlock:
		return int64(len(t.data))
	}
	return 0
}

// ----------------------------------------------------------------------------------------------------------------
// This works well
// ----------------------------------------------------------------------------------------------------------------
// scan the file until deadline.
// write blocks to out.
// write partial block after flushwait
// close out on return
// adjust blocksize and flushwait to favor latency or efficiency.
// larger block size with longer flushwait insures full blocks but adds latency.
//
// tail would use a timeout instead of deadline.

func ScanBlocks(out chan<- interface{}, file *os.File, offset int64, deadline time.Time, blocksize int, flushwait time.Duration) {
	// closes out on return
	defer func() {
		// report
		close(out)
	}()

	file.Seek(offset, os.SEEK_SET)
	info, _ := file.Stat()
	ino, dev := FileSysId(&info)

	fmt.Printf("---ScSTA------------------ INODE: %d @ %d\n", ino, offset)
	scanStart := &ScanStart{info, offset, time.Now()}
	out <- scanStart

	blockmiss := 0 // DEBUG
	blocks := 1    // debug

	boff := 0
	flush := false
	backoff := false
	b, flushdeadline := newBlock(blocksize, flushwait)

	for {
		n, e := file.Read(b[boff:])
		switch {
		case e != nil: // and n is potentially > 0
			// EOF - n >= 0
			if boff > 0 && time.Now().After(flushdeadline) {
				flush = true // times up
			} else if n == 0 {
				// let the writer get ahead a bit ..
				backoff = true
			}
		default:
			flush = true
		}
		boff += n

		if flush {
			// flush block
			out <- &FileBlock{b[:boff], offset}
			offset += int64(boff)

			if boff < blocksize {
				blockmiss++
			}

			boff = 0
			flush = false
			b, flushdeadline = newBlock(blocksize, flushwait)
			blocks++
		}
		if backoff {
			time.Sleep(flushwait / 1)
			backoff = false
		}
		if deadline != time_never && time.Now().After(deadline) {
			//			log.Printf("deadline expire - blocks:%d misses:%d\n", blocks, blockmiss)
			Timestamps := [2]time.Time{scanStart.timestamp, time.Now()}
			Offsets := [2]int64{scanStart.offset, offset}
			scanEnd := &ScanReport{
				file:       info,
				Offsets:    Offsets,
				Inode:      uint64(ino),
				Device:     int64(dev),
				Timestamps: Timestamps,
			}
			out <- scanEnd
			fmt.Printf("---ScEND------------------ INODE: %d @ %d\n", ino, offset)
			fmt.Println("SCAN - file scan end")
			return
		}
	}
}

// yes, its a hack
func newBlock(size int, wait time.Duration) (b []byte, deadline time.Time) {
	return make([]byte, size), time.Now().Add(wait)
}

// ----------------------------------------------------------------------------------------------------------------
////////// make bundles from lines  ////////////////////////////////
// ----------------------------------------------------------------------------------------------------------------

type Bundle struct {
	lines [][]byte
	len   int
}

func NewBundle(size int) *Bundle {
	return &Bundle{make([][]byte, size), 0}
}

func BundleLines(in <-chan []byte, out chan<- *Bundle, size int, flushwait time.Duration) {
	defer close(out)

	bundle := NewBundle(size)
	n := 0
	flush := false
	quit := false
	for {
		select {
		case line, ok := <-in:
			if !ok {
				quit = true
				flush = true
			}
			if line != nil {
				bundle.lines[n] = line
				n++
			}
			if n == size {
				flush = true
			}
		case <-time.After(flushwait):
			if n > 0 {
				//				fmt.Println("bundle flush..")
				flush = true
			}
		}

		if flush {
			bundle.len = n
			out <- bundle
			bundle = NewBundle(size)
			n = 0
			flush = false
		}
		if quit {
			break
		}
	}
}

// ----------------------------------------------------------------------------------------------------------------
////////// scan lines from blocks ////////////////////////////////
// ----------------------------------------------------------------------------------------------------------------

// processes byte blocks from in channel and emits
// complete line []byte to out.
// stops on in clone.
// closes out on return.
func ScanLines(in <-chan []byte, out chan<- []byte) {

	defer close(out)

	var linebuf []byte
	for buf := range in {
		xoff := 0

		// completing a fragment from previous block?
		// try and seek end of line in this block
		if linebuf != nil {
			xoff = seek(buf, byte('\n'))
			if xoff != -1 {
				// complete fragment
				linebuf = append(linebuf, buf[:xoff]...)
				out <- linebuf
				xoff++ // skip the delim
				linebuf = nil
			} else {
				// the entire block is still a fragment
				// just add it and move to next block
				linebuf = append(linebuf, buf...)
				continue // nextblock
			}
		}

		// scan remaining lines in block and fragment (if any)
		lines, frag := scanLines0(buf[xoff:])
		for _, line := range lines {
			out <- line
			//			fmt.Println("scanned line")
		}
		if len(frag) > 0 {
			linebuf = frag
		}
	}
}

// returns the first offset of the first occurrence of delimeter
// or -1 if not found.
func seek(buf []byte, delim byte) int {
	for i, b := range buf {
		if b == delim {
			return i
		}
	}
	return -1
}

// Returns array of slices of the input buf,
// where each slice is a \n delimited line.
// If buf begins with \n, the first slice is zero len
// if buf does not end in a \n delimited line, then the fragment is returned.
//
// overhead: we append the slice array but it should be ok.
func scanLines0(buf []byte) ([][]byte, []byte) {
	var lines [][]byte

	offset := 0
	for i, b := range buf {
		if b == '\n' {
			slice := buf[offset:i]
			lines = append(lines, slice)
			offset = i + 1 // skip delim
		}
	}
	return lines, buf[offset:]
}

//// temp - move to lsf/logfiles or ~/schema in general
//type ScanReport struct {
//	file       os.FileInfo
//	Inode      uint64
//	Device     int64
//	Offsets    [2]int64
//	Timestamps [2]time.Time
//}
//
//func (t *ScanReport) String() string {
//	return fmt.Sprintf("%d %d %d %d %d %d", t.Inode, t.Device, t.Offsets[0], t.Offsets[1], t.Timestamps[0].UnixNano(), t.Timestamps[1].UnixNano())
//}
func LoadScanReportFile(fname string) *ScanReport {

	file, e := os.Open(fname)
	if e != nil {
		return nil
	}
	defer file.Close()

	b := bufio.NewReader(file)
	line, e := b.ReadString('\n')
	if e != nil {
		panic(e)
	}
	scanReport := ReadScanReport(line)
	return scanReport
}

// Given a scan report and ByModTimeAsc, ..
func wot(scanReport *ScanReport, trackingReports []*TrackReport) {

}

// REVU: this should only be called after checking that file hasn't grown.
// Given a scan report and ByModTimeAsc for log pattern determine which file
// and at which offset needs to be scanned.
// Function will find the next youngest file
//
// assumption here is that
//
// a : all of the files in tracking actually belong to stream
//
// b :

var E_NOTHING_TO_TRACK = fmt.Errorf("no candidate for tracking was found")
var E_NO_TRACKING_INFO = fmt.Errorf("no tracking info provided")
var E_LOST_TRACK = fmt.Errorf("lost track of log stream file")

func SameFile(trackReport *TrackReport, scanReport *ScanReport) bool {
	return trackReport.Inode == scanReport.Inode && trackReport.Device == scanReport.Device
}

func DetermineNextScanSpecBasedOnTracking(scanReport *ScanReport, trackingReports []*TrackReport) (file *os.FileInfo, atOffset int64, e error) {
	// if report list is empty, return nil, -1
	if len(trackingReports) == 0 {
		e = E_NO_TRACKING_INFO
		return
	}

	if scanReport == nil {
		fmt.Println("NO SCAN HISTORY - USING OLDEST %s @ 0", trackingReports[0].file)
		return trackingReports[0].file, 0, nil
	}

	scanEndOffset := scanReport.Offsets[1]
	scanEndTime := scanReport.Timestamps[1].Unix()

	sort.Sort(ByModTimeAsc(trackingReports))
	//	fmt.Println("------ SORTED")
	//	for _, tracking0 := range trackingReports {
	//		fmt.Println(tracking0)
	//	}
	//	fmt.Println("------ SORTED END")
	for _, tracking0 := range trackingReports {
		if SameFile(tracking0, scanReport) {
			fmt.Println("CRNT", tracking0)
			//			fmt.Printf("ME >> %d - %d - %s - %s\n", tracking0.Inode, tracking0.ModTime.Unix(), tracking0.ModTime, (*tracking0.file).Name())
			if tracking0.Size > scanEndOffset {
				//				fmt.Printf("PICK SAME - size increased - %s\n", (*tracking0.file).Name())
				return tracking0.file, scanEndOffset, nil
			}
		}
	}
	//	fmt.Println("------")

	for _, tracking := range trackingReports {
		//		fname := (*tracking.file).Name()
		if !SameFile(tracking, scanReport) {
			//			fmt.Printf("candidate: %d %s\n", tracking.ModTime.Unix(), tracking)
			//			fmt.Printf("reference: %d %s\n", scanEndTime, scanReport)
			//			fmt.Printf("candidate: %d\nscanrepor: %d\n", tracking.ModTime.Unix(), scanReport.Timestamps[1].Unix())
			if tracking.ModTime.Unix() >= scanEndTime {
				pick := tracking
				if pick.Size > 0 {
					fmt.Println("PICK", pick)
					return pick.file, 0, nil
					//				} else {
					//					e = E_NOTHING_TO_TRACK
					//					return
				}
			}
		}
	}
	e = E_LOST_TRACK
	return
}

type FileOp func(info *os.FileInfo) (done bool)

func GlobApply(pattern string, function FileOp) {
	matches, e := filepath.Glob(pattern)
	if e != nil {
		panic(e)
	}
	for _, match := range matches {
		info, e := os.Stat(match)
		if e != nil {
			panic(e)
		}
		if done := function(&info); done { // being clear about semantics
			return
		}
	}
}

func DoMatchFileSysId(inode uint64, device int64, match **os.FileInfo) FileOp {
	return func(info *os.FileInfo) bool {
		ino, dev := FileSysId(info)
		if ino == inode && int64(dev) == device {
			*match = info
			return true
		}
		return false
	}
}

func DoGenerateTrackReport(reports *[]*TrackReport) FileOp {
	return func(info *os.FileInfo) bool {
		report := NewTrackReport(info)
		arr := append(*reports, report)
		*reports = arr
		return false
	}
}

// searches in current directory for any file that matches
// the given file system identifiers
//
func FindFileByFileSysId(inode uint64, device int64) *os.FileInfo {
	matches, e := filepath.Glob("*")
	if e != nil {
		panic(e)
	}
	for _, match := range matches {
		info, e := os.Stat(match)
		if e != nil {
			panic(e)
		}
		ino, dev := FileSysId(&info)
		if ino == inode && int64(dev) == device {
			return &info
		}
	}
	return nil
}

//func FileSysId(info *os.FileInfo) (uint64, int32) {
//	fstat := (*info).Sys().(*syscall.Stat_t)
//	//	fmt.Println(fstat)
//	return fstat.Ino, fstat.Dev
//}
//
type TrackReport struct {
	file    *os.FileInfo
	Inode   uint64
	Device  int64
	ModTime time.Time
	Size    int64
	Name    string
}

type ByModTimeAsc []*TrackReport

func (a ByModTimeAsc) Len() int           { return len(a) }
func (a ByModTimeAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByModTimeAsc) Less(i, j int) bool { return a[i].ModTime.Before(a[j].ModTime) }

func NewTrackReport(info *os.FileInfo) *TrackReport {
	inode, device := FileSysId(info)

	report := &TrackReport{
		file:    info,
		Inode:   inode,
		Device:  int64(device),
		ModTime: (*info).ModTime(),
		Size:    (*info).Size(),
		Name:    (*info).Name(),
	}

	return report
}

func (t *TrackReport) String() string {
	return fmt.Sprintf("%d %d %d %d %s", t.Inode, t.Device, t.ModTime.Unix(), t.Size, t.Name)
}

func ReadTrackReport(s string) *TrackReport {
	defer func() {
		p := recover()
		if p != nil {
			panic(p)
		}
	}()
	r := TrackReport{}

	var unixsecs int64
	fmt.Sscanf(s, "%d %d %d %d %s", &r.Inode, &r.Device, &unixsecs, &r.Size, &r.Name)
	r.ModTime = time.Unix(unixsecs, 0)
	return &r
}
