package client

import (
  "io"

  "github.com/TopPano/providence-cli/api/types"
  "golang.org/x/net/context"
)

type CommonAPIClient interface {
  EngineAPIClient
}

// EngineAPIClient defines API client methods for the engines.
type EngineAPIClient interface {
  EngineBuild(ctx context.Context, context io.Reader, options types.EngineBuildOptions) (types.EngineBuildResponse, error)
}

// APIClient is an interface that clients that talk with a Providence server must implement.
type APIClient interface {
  CommonAPIClient
}

// Ensure that Client always implements APIClient.
var _ APIClient = &Client{}
