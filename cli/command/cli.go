package command

import (
  "errors"
  "io"
  "os"

  "github.com/TopPano/providence-cli/api"
  cliflags "github.com/TopPano/providence-cli/cli/flags"
  "github.com/TopPano/providence-cli/client"
)

// Streams is an interface which exposes the standard input and output streams
type Streams interface {
  In()  *InStream
  Out() *OutStream
  Err() io.Writer
}

// ProvCli represents the providence command line client.
// Instances of the client can be returned from NewProvCli.
type ProvCli struct {
  in      *InStream
  out     *OutStream
  err     io.Writer
  client  client.APIClient
}

// Client returns the APIClient
func (cli *ProvCli) Client() client.APIClient {
  return cli.client
}

// Out returns the writer used for stdout
func (cli *ProvCli) Out() *OutStream {
  return cli.out
}

// Err returns the writer used for stderr
func (cli *ProvCli) Err() io.Writer {
  return cli.err
}

// In returns the reader used for stdin
func (cli *ProvCli) In() *InStream {
  return cli.in
}

// Initialize the ProvCli runs initialization that must happen after command
// line flags are parsed.
func (cli *ProvCli) Initialize(opts *cliflags.ClientOptions) error {
  var err error
  cli.client, err = NewAPIClientFromFlags(opts.Common)
  if err != nil {
    return err
  }
  return nil
}

// NewProvCli returns a ProvCli instance with IO output and error streams set by in, out and err.
func NewProvCli(in io.ReadCloser, out, err io.Writer) *ProvCli {
  return &ProvCli{in: NewInStream(in), out: NewOutStream(out), err: err}
}

// NewAPIClientFromFlags creates a new APIClient from command line flags
func NewAPIClientFromFlags(opts *cliflags.CommonOptions) (client.APIClient, error) {
  host, err := getServerHost(opts.Hosts)
  if err != nil {
    return &client.Client{}, err
  }

  customHeaders := map[string]string{}

  customHeaders["User-Agent"] = UserAgent()

  verStr := api.DefaultVersion
  if tmpStr := os.Getenv("PROVIDENCE_API_VERSION"); tmpStr != "" {
    verStr = tmpStr
  }

  return client.NewClient(host, verStr, nil, customHeaders)
}

func getServerHost(hosts []string) (host string, err error) {
  switch len(hosts) {
  case 0:
    host = os.Getenv("PROVIDENCE_HOST")
  case 1:
    host = hosts[0]
  default:
    return "", errors.New("Please specify only one -H")
  }

  return host, nil
}

// UserAgent returns the user agent string used for making API requests.
func UserAgent() string {
  return "Providence-Client/"
}
