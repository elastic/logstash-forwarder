package liblumberjack

import (
  "time"
  "path/filepath"
  "syscall"
  "os"
  "log"
)

func Prospect(paths []string, output chan *FileEvent) {
  // Scan for "-" to do stdin special handling.
  for i, path := range paths {
    if path == "-" {
      harvester := Harvester{Path: path}
      go harvester.Harvest(output)

      // remove "-" from the paths list
      paths = append(paths[0:i], paths[i+1:]...)
      break
    }
  }

  fileinfo := make(map[string]os.FileInfo)
  for {
    for _, path := range paths {
      prospector_scan(path, fileinfo, output)
    }

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable
  }
} /* Prospect */

func prospector_scan(path string, fileinfo map[string]os.FileInfo,
                     output chan *FileEvent) {
  log.Printf("Prospecting %s\n", path)

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
          harvester := Harvester{Path: file}
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
        harvester := Harvester{Path: file}
        go harvester.Harvest(output)
      }
    }
  } // for each file matched by the glob
}
