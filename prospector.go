package main

import (
  "time"
  "path/filepath"
  "encoding/json"
  "os"
  "log"
)

func Prospect(fileconfig FileConfig, output chan *FileEvent) {
  fileinfo := make(map[string]os.FileInfo)

  // Handle any "-" (stdin) paths
  for i, path := range fileconfig.Paths {
    if path == "-" {
      harvester := Harvester{Path: path, Fields: fileconfig.Fields}
      go harvester.Harvest(output)

      // Remove it from the file list
      fileconfig.Paths = append(fileconfig.Paths[:i], fileconfig.Paths[i+1:]...)
    }
  }

  // Use the registrar db to reopen any files at their last positions
  resume_tracking(fileconfig, fileinfo, output)

  for {
    for _, path := range fileconfig.Paths {
      prospector_scan(path, fileconfig.Fields, fileinfo, output)
    }

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable
  }
} /* Prospect */

func resume_tracking(fileconfig FileConfig, fileinfo map[string]os.FileInfo, output chan *FileEvent) {
  // Start up with any registrar data.
  history, err := os.Open(".lumberjack")
  if err == nil {
    historical_state := make(map[string]*FileState)
    log.Printf("Loading registrar data\n")
    decoder := json.NewDecoder(history)
    decoder.Decode(&historical_state)
    history.Close()

    for path, state := range historical_state {
      // if the file is the same inode/device as we last saw,
      // start a harvester on it at the last known position
      info, err := os.Stat(path)
      if err != nil { continue }

      if is_file_same(path, info, state) {
        // same file, seek to last known position
        fileinfo[path] = info

        for _, pathglob := range fileconfig.Paths {
          match, _ := filepath.Match(pathglob, path)
          if match {
            harvester := Harvester{Path: path, Fields: fileconfig.Fields, Offset: state.Offset }
            go harvester.Harvest(output)
            break
          }
        }
      }
    }
  }
}

func prospector_scan(path string, fields map[string]string, 
                     fileinfo map[string]os.FileInfo,
                     output chan *FileEvent) {
  //log.Printf("Prospecting %s\n", path)

  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

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
    // Track the stat data for this file for later comparison to check for
    // rotation/etc
    fileinfo[file] = info

    // Conditions for starting a new harvester:
    // - file path hasn't been seen before
    // - the file's inode or device changed
    if !is_known { 
      // TODO(sissel): Skip files with modification dates older than N
      // TODO(sissel): Make the 'ignore if older than N' tunable
      if time.Since(info.ModTime()) > 24*time.Hour {
        log.Printf("Skipping old file: %s\n", file)
      } else if is_file_renamed(file, info, fileinfo) {
        // Check to see if this file was simply renamed (known inode+dev)
      } else {
        // Most likely a new file. Harvest it!
        log.Printf("Launching harvester on new file: %s\n", file)
        harvester := Harvester{Path: file, Fields: fields}
        go harvester.Harvest(output)
      }
    } else if !is_fileinfo_same(lastinfo, info) {
      log.Printf("Launching harvester on rotated file: %s\n", file)
      // TODO(sissel): log 'file rotated' or osmething
      // Start a harvester on the path; a new file appeared with the same name.
      harvester := Harvester{Path: file, Fields: fields}
      go harvester.Harvest(output)
    }
  } // for each file matched by the glob
}
