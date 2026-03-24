package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script",
	Long: `Generate shell completion script for bash, zsh, or fish.

To load completions:

Bash:
  $ source <(isolate-panel completion bash)
  $ isolate-panel completion bash > /etc/bash_completion.d/isolate-panel

Zsh:
  $ source <(isolate-panel completion zsh)
  $ isolate-panel completion zsh > /usr/local/share/zsh/site-functions/_isolate-panel

Fish:
  $ isolate-panel completion fish | source
  $ isolate-panel completion fish > ~/.config/fish/completions/isolate-panel.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		}
	},
}

// CompletionCmd returns the completion command
func CompletionCmd() *cobra.Command {
	return completionCmd
}

func init() {
	// Completion command is added in main.go
}
