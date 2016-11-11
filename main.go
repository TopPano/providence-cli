package main

import (
  "fmt"
  "os"

  "github.com/Sirupsen/logrus"
  "github.com/TopPano/providence-cli/cli"
  "github.com/TopPano/providence-cli/cli/command"
  "github.com/TopPano/providence-cli/cli/command/commands"
  cliflags "github.com/TopPano/providence-cli/cli/flags"
  "github.com/docker/docker/pkg/term"
  "github.com/dnephin/cobra"
  "github.com/spf13/pflag"
)

func newProvidenceCommand(provCli *command.ProvCli) *cobra.Command {
  opts := cliflags.NewClientOptions()
  var flags *pflag.FlagSet

  cmd := &cobra.Command{
    Use:              "prov [OPTIONS] COMMAND [arg...]",
    Short:            "A self-sufficient runtime for operating with Providence service.",
    SilenceUsage:     true,
    SilenceErrors:    true,
    TraverseChildren: true,
    Args:             noArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
      if opts.Version {
        showVersion()
        return nil
      }
      fmt.Fprintf(provCli.Err(), "\n"+cmd.UsageString())
      return nil
    },
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
      // flags must be the top-level command flags, not cmd.Flags()
      opts.Common.SetDefaultOptions(flags)
      provPreRun(opts)
      return provCli.Initialize(opts)
    },
  }
  cli.SetupRootCommand(cmd)

  flags = cmd.Flags()
  flags.BoolVarP(&opts.Version, "version", "v", false, "Print version information and quit")
  opts.Common.InstallFlags(flags)

  cmd.SetOutput(provCli.Out())
  commands.AddCommands(cmd, provCli)

  return cmd
}

func noArgs(cmd *cobra.Command, args []string) error {
  if len(args) == 0 {
    return nil
  }
  return fmt.Errorf(
    "prov: '%s' is not a providence command.\nSee 'prov --help'%s'", args[0], ".")
}

func main() {
  // Set terminal emulation based on platform as required.
  stdin, stdout, stderr := term.StdStreams()
  logrus.SetOutput(stderr)

  provCli := command.NewProvCli(stdin, stdout, stderr)
  cmd := newProvidenceCommand(provCli)

  if err := cmd.Execute(); err != nil {
    if sterr, ok := err.(cli.StatusError); ok {
      if sterr.Status != "" {
        fmt.Fprintln(stderr, sterr.Status)
      }
      // StatusError should only be used for errors, and all errors should
      // have a non-zero exit status, so never exit with 0
      if sterr.StatusCode == 0 {
        os.Exit(1)
      }
      os.Exit(sterr.StatusCode)
    }
    fmt.Fprintln(stderr, err)
    os.Exit(1)
  }
}

func showVersion() {
  fmt.Printf("Providence version 1.0")
}

func provPreRun(opts *cliflags.ClientOptions) {
  cliflags.SetLogLevel(opts.Common.LogLevel)
}
