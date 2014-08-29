package main

import (
	"os"
	"path/filepath"
	"time"
)

type ProspectorResume struct {
	files   map[string]*FileState
	persist chan *FileState
}

type ProspectorInfo struct {
	fileinfo  os.FileInfo /* the file info */
	harvester chan int64  /* the harvester will send an event with its offset when it closes */
	last_seen uint32      /* int number of the last iterations in which we saw this file */
}

type Prospector struct {
	FileConfig     FileConfig
	prospectorinfo map[string]ProspectorInfo
	iteration      uint32
	lastscan       time.Time
}

func (p *Prospector) Prospect(resume *ProspectorResume, output chan *FileEvent) {
	p.prospectorinfo = make(map[string]ProspectorInfo)

	// Handle any "-" (stdin) paths
	for i, path := range p.FileConfig.Paths {
		if path == "-" {
			// Offset and Initial never get used when path is "-"
			harvester := Harvester{Path: path, FileConfig: p.FileConfig}
			go harvester.Harvest(output)

			// Remove it from the file list
			p.FileConfig.Paths = append(p.FileConfig.Paths[:i], p.FileConfig.Paths[i+1:]...)
		}
	}

	// Seed last scan time
	p.lastscan = time.Now()

	// Now let's do one quick scan to pick up new files
	for _, path := range p.FileConfig.Paths {
		p.scan(path, output, resume)
	}

	// This signals we finished considering the previous state
	event := &FileState{
		Source: nil,
	}
	resume.persist <- event

	for {
		newlastscan := time.Now()

		for _, path := range p.FileConfig.Paths {
			// Scan - flag false so new files always start at beginning
			p.scan(path, output, nil)
		}

		p.lastscan = newlastscan

		// Defer next scan for a bit.
		time.Sleep(10 * time.Second) // Make this tunable

		// Clear out files that disappeared and we've stopped harvesting
		for file, lastinfo := range p.prospectorinfo {
			if len(lastinfo.harvester) != 0 && lastinfo.last_seen < p.iteration {
				delete(p.prospectorinfo, file)
			}
		}

		p.iteration++ // Overflow is allowed
	}
} /* Prospect */

func (p *Prospector) scan(path string, output chan *FileEvent, resume *ProspectorResume) {

	// Evaluate the path as a wildcards/shell glob
	matches, err := filepath.Glob(path)
	if err != nil {
		emit("glob(%s) failed: %v\n", path, err)
		return
	}

	// To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
	missinginfo := make(map[string]os.FileInfo)

	// Check any matched files to see if we need to start a harvester
	for _, file := range matches {
		// Stat the file, following any symlinks.
		fileinfo, err := os.Stat(file)
		// TODO(sissel): check err
		if err != nil {
			emit("stat(%s) failed: %s\n", file, err)
			continue
		}

		if fileinfo.IsDir() {
			emit("Skipping directory: %s\n", file)
			continue
		}

		// Check the current info against p.prospectorinfo[file]
		lastinfo, is_known := p.prospectorinfo[file]
		newinfo := lastinfo

		// Conditions for starting a new harvester:
		// - file path hasn't been seen before
		// - the file's inode or device changed
		if !is_known {
			// Create a new prospector info with the stat info for comparison
			newinfo = ProspectorInfo{fileinfo: fileinfo, harvester: make(chan int64, 1), last_seen: p.iteration}

			// Check for dead time, but only if the file modification time is before the last scan started
			// This ensures we don't skip genuine creations with dead times less than 10s
			if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > p.FileConfig.deadtime {
				var offset int64 = 0
				var is_resuming bool = false

				if resume != nil {
					// Call the calculator - it will process resume state if there is one
					offset, is_resuming = p.calculate_resume(file, fileinfo, resume)
				}

				// Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
				// This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
				// Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
				if is_resuming {
					emit("Resuming harvester on a previously harvested file: %s\n", file)
					harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.harvester}
					go harvester.Harvest(output)
				} else {
					// Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
					emit("Skipping file (older than dead time of %v): %s\n", p.FileConfig.deadtime, file)
					newinfo.harvester <- fileinfo.Size()
				}
			} else if previous := is_file_renamed(file, fileinfo, p.prospectorinfo, missinginfo); previous != "" {
				// This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
				emit("File rename was detected: %s -> %s\n", previous, file)

				newinfo.harvester = p.prospectorinfo[previous].harvester
			} else {
				var offset int64 = 0
				var is_resuming bool = false

				if resume != nil {
					// Call the calculator - it will process resume state if there is one
					offset, is_resuming = p.calculate_resume(file, fileinfo, resume)
				}

				// Are we resuming a file or is this a completely new file?
				if is_resuming {
					emit("Resuming harvester on a previously harvested file: %s\n", file)
				} else {
					emit("Launching harvester on new file: %s\n", file)
				}

				// Launch the harvester
				harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.harvester}
				go harvester.Harvest(output)
			}
		} else {
			// Update the fileinfo information used for future comparisons, and the last_seen counter
			newinfo.fileinfo = fileinfo
			newinfo.last_seen = p.iteration

			if !is_fileinfo_same(lastinfo.fileinfo, fileinfo) {
				if previous := is_file_renamed(file, fileinfo, p.prospectorinfo, missinginfo); previous != "" {
					// This file was renamed from another file we know - link the same harvester channel as the old file
					emit("File rename was detected: %s -> %s\n", previous, file)
					emit("Launching harvester on renamed file: %s\n", file)

					newinfo.harvester = p.prospectorinfo[previous].harvester
				} else {
					// File is not the same file we saw previously, it must have rotated and is a new file
					emit("Launching harvester on rotated file: %s\n", file)

					// Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
					newinfo.harvester = make(chan int64, 1)

					// Start a harvester on the path
					harvester := &Harvester{Path: file, FileConfig: p.FileConfig, FinishChan: newinfo.harvester}
					go harvester.Harvest(output)
				}

				// Keep the old file in missinginfo so we don't rescan it if it was renamed and we've not yet reached the new filename
				// We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
				missinginfo[file] = lastinfo.fileinfo
			} else if len(newinfo.harvester) != 0 && lastinfo.fileinfo.ModTime() != fileinfo.ModTime() {
				// Resume harvesting of an old file we've stopped harvesting from
				emit("Resuming harvester on an old file that was just modified: %s\n", file)

				// Start a harvester on the path; an old file was just modified and it doesn't have a harvester
				// The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
				harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: <-newinfo.harvester, FinishChan: newinfo.harvester}
				go harvester.Harvest(output)
			}
		}

		// Track the stat data for this file for later comparison to check for
		// rotation/etc
		p.prospectorinfo[file] = newinfo
	} // for each file matched by the glob
}

func (p *Prospector) calculate_resume(file string, fileinfo os.FileInfo, resume *ProspectorResume) (int64, bool) {
	last_state, is_found := resume.files[file]

	if is_found && is_file_same(file, fileinfo, last_state) {
		// We're resuming - throw the last state back downstream so we resave it
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		resume.persist <- last_state
		return last_state.Offset, true
	}

	if previous := is_file_renamed_resumelist(file, fileinfo, resume.files); previous != "" {
		// File has rotated between shutdown and startup
		// We return last state downstream, with a modified event source with the new file name
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		emit("Detected rename of a previously harvested file: %s -> %s\n", previous, file)
		last_state := resume.files[previous]
		last_state.Source = &file
		resume.persist <- last_state
		return last_state.Offset, true
	}

	if is_found {
		emit("Not resuming rotated file: %s\n", file)
	}

	// New file so just start from an automatic position
	return 0, false
}
