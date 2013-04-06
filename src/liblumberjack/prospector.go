package liblumberjack

import (
  "time"
  "path/filepath"
  "fmt"
  "syscall"
  "os"
)

func Prospect(paths []string, output chan *FileEvent) {
  fileinfo := make(map[string]os.FileInfo)

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

  for {
    for _, path := range paths {
      // Evaluate the path as a wildcards/shell glob
      matches, err := filepath.Glob(path)
      if err != nil {
        fmt.Print("glob(%s) failed: %v\n", path, err)
        continue
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
          fmt.Printf("stat(%s) failed: %s\n", file, err)
          continue
        }

        if info.IsDir() {
          // TODO(sissel): log 'skipping directory'
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
          // TODO(sissel): log 'new file found'
          harvester := Harvester{Path: file}
          go harvester.Harvest(output)
        } else {
          // TODO(sissel): FileInfo.Sys() can be nil on unsupported platforms.
          laststat := lastinfo.Sys().(*syscall.Stat_t)
          stat := info.Sys().(*syscall.Stat_t)
          // Compare inode and device; it's a 'new file' if either have changed.
          // aka, the file was rotated/renamed/whatever
          if stat.Dev != laststat.Dev || stat.Ino != laststat.Ino {
            // TODO(sissel): log 'file rotated' or osmething
            // Start a harvester on the path; a new file appeared with the same name.
            harvester := Harvester{Path: file}
            go harvester.Harvest(output)
          }
        }
      } // for each file matched by the glob
    } // for each path in the paths

    // Make this tunable
    time.Sleep(10 * time.Second)
  } // forever
} /* Prospect */
