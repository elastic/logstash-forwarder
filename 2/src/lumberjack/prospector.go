package lumberjack

import (
  "time"
  "path/filepath"
  "fmt"
)


func Prospect(paths []string, output chan *FileEvent) {
  // For each path
  //   - evaluate glob
  //   - for any new file paths, start a harvester

  active := make(map[string]Harvester)

  for {
    for _, path := range paths {
      matches, err := filepath.Glob(path)
      if err != nil {
        fmt.Print("glob(%s) failed: %v\n", path, err)
        continue
      }

      for _, file := range matches {
        // Skip already-watched files
        if _, already_exists := active[file]; already_exists { continue }

        harvester := Harvester{Path: file}
        active[file] = harvester
        go harvester.Harvest(output)
      } // for each file matched by the glob
    } // for each path in the paths

    time.Sleep(10 * time.Second)
  } // forever
} /* Prospect */
