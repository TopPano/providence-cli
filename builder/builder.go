// Package builder defines interfaces for any Providence builder to implement.
package builder

import (
  "io"
  "os"
)

const (
  // DefaultEnginefileName is the Default filename with Engine commands, read by `prov engine build`
  DefaultEnginefileName string = "Enginefile"
)

// Context represents a file system tree.
type Context interface {
  // Close allows to signal that the filesystem tree won't be used anymore.
  // For Context implementations using a temporary directory, it is recommended to
  // delete the temporary directory in Close().
  Close() error
  // Stat returns an entry corresponding to path if any.
  // It is recommended to return an error if path was not found.
  // If path is a symlink it also returns the path to the target file.
  Stat(path string) (string, FileInfo, error)
  // Open opens path from the context and returns a readable stream of it.
  Open(path string) (io.ReadCloser, error)
  // Walk walks the tree of the context with the function passed to it.
  Walk(root string, walkFn WalkFunc) error
}

// WalkFunc is the type of the function called for each file or directory visited by Context.Walk().
type WalkFunc func(path string, fi FileInfo, err error) error

// ModifiableContext represents a modifiable Context.
// TODO: remove this interface once we can get rid of Remove()
type ModifiableContext interface {
  Context
  // Remove deletes the entry specified by `path`.
  // It is usual for directory entries to delete all its subentries.
  Remove(path string) error
}

// FileInfo extends os.FileInfo to allow retrieving an absolute path to the file.
// TODO: remove this interface once pkg/archive exposes a walk function that Context can use.
type FileInfo interface {
  os.FileInfo
  Path() string
}

// PathFileInfo is a convenience struct that implements the FileInfo interface.
type PathFileInfo struct {
  os.FileInfo
  // FilePath holds the absolute path to the file.
  FilePath string
  // Name holds the basename for the file.
  FileName string
}

// Path returns the absolute path to the file.
func (fi PathFileInfo) Path() string {
  return fi.FilePath
}

// Name returns the basename of the file.
func (fi PathFileInfo) Name() string {
  if fi.FileName != "" {
    return fi.FileName
  }
  return fi.FileInfo.Name()
}

// Hashed defines an extra method intended for implementations of os.FileInfo.
type Hashed interface {
  // Hash returns the hash of a file.
  Hash() string
  SetHash(string)
}

// HashedFileInfo is a convenient struct that augments FileInfo with a field.
type HashedFileInfo struct {
  FileInfo
  // FileHash represents the hash of a file.
  FileHash string
}

// Hash returns the hash of a file.
func (fi HashedFileInfo) Hash() string {
  return fi.FileHash
}

// SetHash sets the hash of a file.
func (fi *HashedFileInfo) SetHash(h string) {
  fi.FileHash = h
}

