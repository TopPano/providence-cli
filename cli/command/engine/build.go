package engine

import (
  "bytes"
  "fmt"
  "io"
  "os"
  "path/filepath"

  "golang.org/x/net/context"

  "github.com/TopPano/providence-cli/api/types"
  "github.com/TopPano/providence-cli/builder"
  "github.com/TopPano/providence-cli/builder/provignore"
  "github.com/TopPano/providence-cli/cli"
  "github.com/TopPano/providence-cli/cli/command"
  "github.com/docker/docker/pkg/archive"
  "github.com/docker/docker/pkg/jsonmessage"
  "github.com/docker/docker/pkg/progress"
  "github.com/docker/docker/pkg/streamformatter"
  "github.com/docker/docker/pkg/urlutil"
  "github.com/dnephin/cobra"
)

type buildOptions struct {
  context         string
  enginefileName  string
  quiet           bool
  compress        bool
}

// NewBuildCommand creates a new `prov engine build` command
func NewBuildCommand(provCli *command.ProvCli) *cobra.Command {
  options := buildOptions{}

  cmd := &cobra.Command{
    Use:    "build [OPTIONS] PATH | URL | -",
    Short:  "Build an engine",
    Args:   cli.ExactArgs(1),
    RunE:   func(cmd *cobra.Command, args []string) error {
      options.context = args[0]
      return runBuild(provCli, options)
    },
  }

  flags := cmd.Flags()

  flags.StringVarP(&options.enginefileName, "file", "f", "", "Name of the Enginefile (Default is 'PATH/Enginefile')")
  flags.BoolVarP(&options.compress, "quiet", "q", false, "Suppress the build output and print engine ID on success")
  flags.BoolVar(&options.compress, "compress", true, "Compress the build context using gzip")

  return cmd
}

// lastProgressOutput is the same as progress.Output except
// that it only output with the last update. It is used in
// non terminal scenarios to depress verbose messages.
type lastProgressOutput struct {
  output progress.Output
}

// WriteProgress formats progress information from a ProgressReader.
func (out *lastProgressOutput) WriteProgress(prog progress.Progress) error {
  if !prog.LastUpdate {
    return nil
  }

  return out.output.WriteProgress(prog)
}

func runBuild(provCli *command.ProvCli, options buildOptions) error {

  var (
    buildCtx  io.ReadCloser
    err       error
  )

  specifiedContext := options.context

  var (
    contextDir    string
    relEnginefile string
    progBuff      io.Writer
    buildBuff     io.Writer
  )

  progBuff = provCli.Out()
  buildBuff = provCli.Out()
  if options.quiet {
    progBuff = bytes.NewBuffer(nil)
    buildBuff = bytes.NewBuffer(nil)
  }

  contextDir, relEnginefile, err = builder.GetContextFromLocalDir(specifiedContext, options.enginefileName)
  if err != nil {
    if options.quiet && urlutil.IsURL(specifiedContext) {
      fmt.Fprintln(provCli.Err(), progBuff)
    }
    return fmt.Errorf("unable to prepare context: %s", err)
  }

  f, err := os.Open(filepath.Join(contextDir, ".provignore"))
  if err != nil && !os.IsNotExist(err) {
    return err
  }
  defer f.Close()

  var excludes []string
  if err == nil {
    excludes, err = provignore.ReadAll(f)
    if err != nil {
      return err
    }
  }

  if err := builder.ValidateContextDirectory(contextDir, excludes); err != nil {
    return fmt.Errorf("Error checking context: '%s'.", err)
  }

  var includes = []string{"."}

  compression := archive.Uncompressed
  if options.compress {
    compression = archive.Gzip
  }
  buildCtx, err = archive.TarWithOptions(contextDir, &archive.TarOptions{
    Compression:      compression,
    ExcludePatterns:  excludes,
    IncludeFiles:     includes,
  })
  if err != nil {
    return err
  }

  ctx := context.Background()

  // Setup an upload progress bar
  progressOutput := streamformatter.NewStreamFormatter().NewProgressOutput(progBuff, true)

  var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Providencer server")

  buildOptions := types.EngineBuildOptions{
    Enginefile:   relEnginefile,
  }

  response, err := provCli.Client().EngineBuild(ctx, body, buildOptions)
  if err != nil {
    if options.quiet {
      fmt.Fprintf(provCli.Err(), "%s", progBuff)
    }
    return err
  }

  defer response.Body.Close()

  err = jsonmessage.DisplayJSONMessagesStream(response.Body, buildBuff, provCli.Out().FD(), provCli.Out().IsTerminal(), nil)
  if err != nil {
    if jerr, ok := err.(*jsonmessage.JSONError); ok {
      // if no error code is set, default to 1
      if jerr.Code == 0 {
        jerr.Code = 1
      }
      if options.quiet {
        fmt.Fprintf(provCli.Err(), "%s%s", progBuff, buildBuff)
      }
      return cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
    }
  }

  // Everything worked so if -q was provided the output from the server
  // should be just the engine ID and we'll print that to stdout.
  if options.quiet {
    fmt.Fprintf(provCli.Out(), "%s", buildBuff)
  }

  return nil
}
