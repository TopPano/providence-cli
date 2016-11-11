package client

import (
  "io"
  "net/http"
  "net/url"

  "golang.org/x/net/context"

  "github.com/TopPano/providence-cli/api/types"
)

// EngineBuild sends request to the server to build engines.
// The Body in the response implement an io.ReadCloser and it's up to the caller to
// close it.
func (cli *Client) EngineBuild(ctx context.Context, buildContext io.Reader, options types.EngineBuildOptions) (types.EngineBuildResponse, error) {
  query, err := engineBuildOptionsToQuery(options)
  if err != nil {
    return types.EngineBuildResponse{}, err
  }

  headers := http.Header(make(map[string][]string))
  headers.Set("Content-Type", "application/tar")

  serverResp, err := cli.postRaw(ctx, "/engine", query, buildContext, headers)
  if err != nil {
    return types.EngineBuildResponse{}, err
  }

  return types.EngineBuildResponse{
    Body: serverResp.body,
  }, nil
}

func engineBuildOptionsToQuery(options types.EngineBuildOptions) (url.Values, error) {
  query := url.Values{}

  query.Set("enginefile", options.Enginefile)

  return query, nil
}
