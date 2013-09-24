package main

import (
  "time"
  "path/filepath"
  "encoding/json"
  "syscall"
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
  history, err := os.Open(appconfig.RegistrarFile)
  if err == nil {
    historical_state := make(map[string]*FileState)
    log.Printf("Loading registrar data %s\n", fileconfig.Paths)
    decoder := json.NewDecoder(history)
    decoder.Decode(&historical_state)
    history.Close()

    for path, state := range historical_state {
      // if the file is the same inode/device as we last saw,
      // start a harvester on it at the last known position
      info, err := os.Stat(path)
      if err != nil { continue }

      fstat := info.Sys().(*syscall.Stat_t)
      if fstat.Ino == state.Inode && fstat.Dev == state.Device {
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
      // Skip files that are too old.  "-1" is never skip.
      if appconfig.IgnoreAfter!=-1 && (time.Since(info.ModTime()) > time.Duration(appconfig.IgnoreAfter) * time.Hour) {
       log.Printf("Skipping file (older than %s): %s\n", time.Duration(appconfig.IgnoreAfter) * time.Hour, file)
      } else {
        // Check to see if this file was simply renamed (known inode+dev)
        stat := info.Sys().(*syscall.Stat_t)
        renamed := false

        for kf, ki := range fileinfo {
          if kf == file {
            continue
          }
          ks := ki.Sys().(*syscall.Stat_t)
          if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
            log.Printf("Skipping %s (old known name: %s)\n", file, kf)
            renamed = true
            // Delete the old entry
            delete(fileinfo, kf)
            break
          }
        }

        if !renamed {
          log.Printf("Launching harvester on new file: %s\n", file)
          harvester := Harvester{Path: file, Fields: fields}
          go harvester.Harvest(output)
        }
      }
    } else {
      // TODO(sissel): FileInfo.Sys() can be nil on unsupported platforms.
      laststat := lastinfo.Sys().(*syscall.Stat_t)
      stat := info.Sys().(*syscall.Stat_t)
      // Compare inode and device; it's a 'new file' if either have changed.
      // aka, the file was rotated/renamed/whatever
      if stat.Dev != laststat.Dev || stat.Ino != laststat.Ino {
        log.Printf("Launching harvester on rotated file: %s\n", file)
        // TODO(sissel): log 'file rotated' or osmething
        // Start a harvester on the path; a new file appeared with the same name.
        harvester := Harvester{Path: file, Fields: fields}
        go harvester.Harvest(output)
      }
    }
  } // for each file matched by the glob
}
