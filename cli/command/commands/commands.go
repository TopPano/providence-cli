package commands

import (
  "github.com/TopPano/providence-cli/cli/command"
  "github.com/TopPano/providence-cli/cli/command/engine"
  "github.com/dnephin/cobra"
)

// AddCommands adds all the commands from cli/command to the root command
func AddCommands(cmd *cobra.Command, provCli *command.ProvCli) {
  cmd.AddCommand(
    engine.NewEngineCommand(provCli),
  )
}
