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

func Prospect(fileconfig FileConfig, resumelist *ProspectorResume, output chan *FileEvent) {
  fileinfo := make(map[string]ProspectorInfo)

  // Handle any "-" (stdin) paths
  for i, path := range fileconfig.Paths {
    if path == "-" {
      harvester := Harvester{Path: path, FileConfig: fileconfig}
      go harvester.Harvest(output)

      // Remove it from the file list
      fileconfig.Paths = append(fileconfig.Paths[:i], fileconfig.Paths[i+1:]...)
    }
  }

  // Now let's do one quick scan to pick up new files - flag true so new files obey from-beginning
  var iteration uint32 = 0
  for _, path := range fileconfig.Paths {
    prospector_scan(iteration, path, fileconfig, fileinfo, output, resumelist)
  }

  // This signals we finished considering the previous state
  event := &FileState{
    Source: nil,
  }
  resumelist.resave <- event

  for {
    for _, path := range fileconfig.Paths {
      // Scan - flag false so new files always start at beginning
      prospector_scan(iteration, path, fileconfig, fileinfo, output, nil)
    }

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable

    // Clear out files that disappeared
    for file, lastinfo := range fileinfo {
      if lastinfo.last_seen < iteration {
        log.Printf("No longer tracking file that hasn't been seen for a while: %s\n", file)
        delete(fileinfo, file)
      }
    }

    iteration++ // Overflow is allowed
  }
} /* Prospect */

// NOTE(driskell): I know, I know... parameter soup... my bad... needs restructuring at some point
func prospector_scan(iteration uint32, path string, fileconfig FileConfig,
  fileinfo map[string]ProspectorInfo, output chan *FileEvent, resumelist *ProspectorResume) {
  //log.Printf("Prospecting %s\n", path)

  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

  // To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
  missingfiles := make(map[string]os.FileInfo)

  // If the glob matches nothing, use the path itself as a literal.
  if len(matches) == 0 && path == "-" {
    matches = append(matches, path)
  }

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

    // Check the current info against fileinfo[file]
    lastinfo, is_known := fileinfo[file]
    newinfo := lastinfo

    // Conditions for starting a new harvester:
    // - file path hasn't been seen before
    // - the file's inode or device changed
    if !is_known {
      // Create a new prospector info with the stat info for comparison
      newinfo = ProspectorInfo{fileinfo: info, harvester: make(chan int64, 1), last_seen: iteration}

      if time.Since(info.ModTime()) > fileconfig.deadtime {
        // If we're in initial run and saved this same files state, push the last offset
        if resumelist == nil {
          // Old file, skip it, but push offset of 0 so we obey from_beginning if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", fileconfig.deadtime, file)
          newinfo.harvester <- 0
        } else if last_state, is_found := resumelist.files[file]; is_found && is_file_same(file, info, last_state) {
          // NOTE(driskell): The logic in prospector could possibly be smoothed out so we don't end up repeating this
          // Even if the file is old, if it's one we're resuming we MUST run harvester so we can ensure we have all entries
          // Once we finish all entries we will timeout read, notice the modification, and end the harvester automatically
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          harvester := &Harvester{Path: file, FileConfig: fileconfig, Offset: last_state.Offset, FinishChan: newinfo.harvester}
          go harvester.Harvest(output)

          resumelist.resave <- last_state
        } else if previous := is_file_renamed_resumelist(file, info, resumelist.files); previous != "" {
          log.Printf("Detected rotation on a previously harvested file: %s -> %s\n", previous, file)
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          harvester := &Harvester{Path: file, FileConfig: fileconfig, Offset: resumelist.files[previous].Offset, FinishChan: newinfo.harvester}
          go harvester.Harvest(output)

          // Since we've renamed we need to modify the event source before passing back to resave
          event := resumelist.files[previous]
          // Copy path since it will change in the loop
          event_source := file
          event.Source = &event_source
          resumelist.resave <- event
        } else {
          // Old file, skip it, but push offset of 0 so we obey from_beginning if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", fileconfig.deadtime, file)
          newinfo.harvester <- 0
        }
      } else if previous := is_file_renamed(file, info, fileinfo, missingfiles); previous != "" {
        // This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
        log.Printf("File rename was detected: %s -> %s\n", previous, file)

        newinfo.harvester = fileinfo[previous].harvester
      } else {
        var offset int64

        // Check if we should resume if we're initial scan, and check by inode too etc in case a rotation occurred while we were stopped
        if resumelist == nil {
          log.Printf("Launching harvester on new file: %s\n", file)
          offset = 0
        } else if last_state, is_found := resumelist.files[file]; is_found && is_file_same(file, info, last_state) {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          offset = last_state.Offset

          resumelist.resave <- last_state
        } else if previous := is_file_renamed_resumelist(file, info, resumelist.files); previous != "" {
          log.Printf("File rename was detected: %s -> %s\n", previous, file)
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          offset = resumelist.files[previous].Offset

          // Since we've renamed we need to modify the event source before passing back to resave
          event := resumelist.files[previous]
          // Copy path since it will change in the loop
          event_source := file
          event.Source = &event_source
          resumelist.resave <- event
        } else {
          log.Printf("Launching harvester on new file: %s\n", file)
          offset = 0
        }

        harvester := &Harvester{Path: file, FileConfig: fileconfig, Offset: offset, FinishChan: newinfo.harvester}
        go harvester.Harvest(output)
      }
    } else {
      // Update the fileinfo information used for future comparisons, and the last_seen counter
      newinfo.fileinfo = info
      newinfo.last_seen = iteration

      // NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
      if !is_fileinfo_same(lastinfo.fileinfo, info) {
        if previous := is_file_renamed(file, info, fileinfo, missingfiles); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Printf("File rename was detected: %s -> %s\n", previous, file)

          newinfo.harvester = fileinfo[previous].harvester
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Printf("Launching harvester on rotated file: %s\n", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          newinfo.harvester = make(chan int64, 1)

          // Start a harvester on the path
          harvester := &Harvester{Path: file, FileConfig: fileconfig, FinishChan: newinfo.harvester}
          go harvester.Harvest(output)
        }

        // Keep the old file in missingfiles so we don't rescan it if it was renamed and we've not yet reached the new filename
        // We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
        missingfiles[file] = lastinfo.fileinfo
      } else if len(newinfo.harvester) != 0 && time.Since(info.ModTime()) < fileconfig.deadtime {
        // NOTE(driskell): If dead time is less than the prospector interval, this stops working
        // Resume harvesting of an old file we've stopped harvesting from
        log.Printf("Resuming harvester on an old file that was just modified: %s\n", file)

        // Start a harvester on the path; an old file was just modified and it doesn't have a harvester
        // The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
        harvester := &Harvester{Path: file, FileConfig: fileconfig, Offset: <-newinfo.harvester, FinishChan: newinfo.harvester}
        go harvester.Harvest(output)
      }
    }

    // Track the stat data for this file for later comparison to check for
    // rotation/etc
    fileinfo[file] = newinfo
  } // for each file matched by the glob
}
