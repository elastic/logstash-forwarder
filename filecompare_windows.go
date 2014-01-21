package main

import (
  "os"
  "reflect"
)

func is_file_same(path string, info os.FileInfo, state *FileState) bool {
  
  // Get details
  idxhi, idxlo, vol := FileIdentifiers(info)
  return idxhi == state.Idxhi && idxlo == state.Idxlo && vol == state.Vol 
}

func is_fileinfo_same(a os.FileInfo, b os.FileInfo) bool {  
  return os.SameFile(a, b)
}

func is_file_renamed(file string, info os.FileInfo, fileinfo map[string]os.FileInfo) bool {

  // Get details
  thisIdxhi, thisIdxlo, thisVol := FileIdentifiers(info)
  
  for kf, ki := range fileinfo {
    if kf == file {
      continue
    }

    thatIdxhi, thatIdxlo, thatVol := FileIdentifiers(ki)

    if thisIdxhi == thatIdxhi && thisIdxlo == thatIdxlo && thisVol == thatVol {
      return true
    }
  }
  
  return false
}

func FileIdentifiers(info os.FileInfo) (uint64, uint64, uint64) {

  value := reflect.ValueOf(info).Elem() // Elem() as it's a pointer
  idxhiField := value.FieldByName("idxhi")
  idxloField := value.FieldByName("idxlo")
  volField := value.FieldByName("vol")
  return idxhiField.Uint(), idxloField.Uint(), volField.Uint()
}
