package engine

import (
  "fmt"

  "github.com/dnephin/cobra"

  "github.com/TopPano/providence-cli/cli"
  "github.com/TopPano/providence-cli/cli/command"
)

// NewEngineCommand returns a cobra command for `engine` subcommands
func NewEngineCommand(provCli *command.ProvCli) *cobra.Command {
  cmd := &cobra.Command{
    Use:    "engine",
    Short:  "Manage engines",
    Args:   cli.NoArgs,
    Run: func(cmd *cobra.Command, args []string) {
      fmt.Fprintf(provCli.Err(), "\n"+cmd.UsageString())
    },
  }
  cmd.AddCommand(
    NewBuildCommand(provCli),
  )

  return cmd
}
