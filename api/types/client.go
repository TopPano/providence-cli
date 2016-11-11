package types

import (
  "io"
)

// EngineBuildOptions holds the information
// necessary to build engines.
type EngineBuildOptions struct {
  Enginefile  string
}

// EngineBuildResponse holds information
// returned by a server after building
// an engine.
type EngineBuildResponse struct {
  Body    io.ReadCloser
}
