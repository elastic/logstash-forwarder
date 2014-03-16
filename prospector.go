package main

import (
  "log"
  "os"
  "path/filepath"
  "time"
)

type ProspectorResume struct {
  files  map[string]*FileState
  resave  chan *FileState
}

type ProspectorInfo struct {
  fileinfo os.FileInfo /* the file info */
  harvester chan int64 /* the harvester will send an event with its offset when it closes */
  last_seen uint32 /* int number of the last iterations in which we saw this file */
}

type Prospector struct {
  FileConfig FileConfig
  fileinfo   map[string]ProspectorInfo
  iteration  uint32
}

func (p *Prospector) Prospect(resumelist *ProspectorResume, output chan *FileEvent) {
  p.fileinfo = make(map[string]ProspectorInfo)

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

  // Now let's do one quick scan to pick up new files - flag true so new files obey from-beginning
  for _, path := range p.FileConfig.Paths {
    p.scan(path, output, resumelist)
  }

  // This signals we finished considering the previous state
  event := &FileState{
    Source: nil,
  }
  resumelist.resave <- event

  for {
    for _, path := range p.FileConfig.Paths {
      // Scan - flag false so new files always start at beginning
      p.scan(path, output, nil)
    }

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable

    // Clear out files that disappeared
    for file, lastinfo := range p.fileinfo {
      if lastinfo.last_seen < p.iteration {
        log.Printf("No longer tracking file that hasn't been seen for a while: %s\n", file)
        delete(p.fileinfo, file)
      }
    }

    p.iteration++ // Overflow is allowed
  }
} /* Prospect */

func (p *Prospector) scan(path string, output chan *FileEvent, resumelist *ProspectorResume) {
  //log.Printf("Prospecting %s\n", path)

  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

  // To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
  missingfiles := make(map[string]os.FileInfo)

  // Check any matched files to see if we need to start a harvester
  for _, file := range matches {
    // Stat the file, following any symlinks.
    info, err := os.Stat(file)
    // TODO(sissel): check err
    if err != nil {
      log.Printf("stat(%s) failed: %s\n", file, err)
      continue
    }

    if info.IsDir() {
      log.Printf("Skipping directory: %s\n", file)
      continue
    }

    // Check the current info against p.fileinfo[file]
    lastinfo, is_known := p.fileinfo[file]
    newinfo := lastinfo

    // Conditions for starting a new harvester:
    // - file path hasn't been seen before
    // - the file's inode or device changed
    if !is_known {
      // Create a new prospector info with the stat info for comparison
      newinfo = ProspectorInfo{fileinfo: info, harvester: make(chan int64, 1), last_seen: p.iteration}

      if time.Since(info.ModTime()) > p.FileConfig.deadtime {
        // Call the calculator - it will process resume state if there is one
        offset, is_resuming := p.calculate_resume(file, info, resumelist)

        // Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
        // This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
        // Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.harvester}
          go harvester.Harvest(output)
        } else {
          // Old file, skip it, but push offset of 0 so we obey from_beginning if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", p.FileConfig.deadtime, file)
          newinfo.harvester <- 0
        }
      } else if previous := is_file_renamed(file, info, p.fileinfo, missingfiles); previous != "" {
        // This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
        log.Printf("File rename was detected: %s -> %s\n", previous, file)

        newinfo.harvester = p.fileinfo[previous].harvester
      } else {
        // Call the calculator - it will process resume state if there is one
        offset, is_resuming := p.calculate_resume(file, info, resumelist)

        // Are we resuming a file or is this a completely new file?
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
        } else {
          log.Printf("Launching harvester on new file: %s\n", file)
        }

        // Launch the harvester
        harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.harvester}
        go harvester.Harvest(output)
      }
    } else {
      // Update the fileinfo information used for future comparisons, and the last_seen counter
      newinfo.fileinfo = info
      newinfo.last_seen = p.iteration

      if !is_fileinfo_same(lastinfo.fileinfo, info) {
        if previous := is_file_renamed(file, info, p.fileinfo, missingfiles); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Printf("File rename was detected: %s -> %s\n", previous, file)
          log.Printf("Launching harvester on renamed file: %s\n", file)

          newinfo.harvester = p.fileinfo[previous].harvester
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Printf("Launching harvester on rotated file: %s\n", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          newinfo.harvester = make(chan int64, 1)

          // Start a harvester on the path
          harvester := &Harvester{Path: file, FileConfig: p.FileConfig, FinishChan: newinfo.harvester}
          go harvester.Harvest(output)
        }

        // Keep the old file in missingfiles so we don't rescan it if it was renamed and we've not yet reached the new filename
        // We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
        missingfiles[file] = lastinfo.fileinfo
      } else if len(newinfo.harvester) != 0 && time.Since(info.ModTime()) < p.FileConfig.deadtime {
        // NOTE(driskell): If dead time is less than the prospector interval, this stops working
        // Resume harvesting of an old file we've stopped harvesting from
        log.Printf("Resuming harvester on an old file that was just modified: %s\n", file)

        // Start a harvester on the path; an old file was just modified and it doesn't have a harvester
        // The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
        harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: <-newinfo.harvester, FinishChan: newinfo.harvester}
        go harvester.Harvest(output)
      }
    }

    // Track the stat data for this file for later comparison to check for
    // rotation/etc
    p.fileinfo[file] = newinfo
  } // for each file matched by the glob
}

func (p *Prospector) calculate_resume(file string, info os.FileInfo, resumelist *ProspectorResume) (int64, bool) {
  if resumelist != nil {
    if last_state, is_found := resumelist.files[file]; is_found && is_file_same(file, info, last_state) {
      // We're resuming - throw the last state back downstream so we resave it
      // And return the offset - also force harvest in case the file is old and we're about to skip it
      resumelist.resave <- last_state
      return last_state.Offset, true
    }

    if previous := is_file_renamed_resumelist(file, info, resumelist.files); previous != "" {
      // File has rotated between shutdown and startup
      // We return last state downstream, with a modified event source with the new file name
      // And return the offset - also force harvest in case the file is old and we're about to skip it
      log.Printf("Detected rotation on a previously harvested file: %s -> %s\n", previous, file)
      event := resumelist.files[previous]
      event.Source = &file
      resumelist.resave <- event
      return event.Offset, true
    }
  }

  // New file so just start from an automatic position if initial scan, or the beginning if subsequent scans
  // The caller will know which to do
  return 0, false
}
