package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/cargo"
	"github.com/yyYank/goodbye/internal/gotools"
	"github.com/yyYank/goodbye/internal/pnpm"
	"github.com/yyYank/goodbye/internal/transfer"
	"github.com/yyYank/goodbye/internal/uv"
)

func init() {
	for _, tool := range []struct {
		name                string
		export, importTools func(transfer.Options) error
	}{
		{"pnpm", pnpm.Export, pnpm.Import},
		{"go", gotools.Export, gotools.Import},
		{"cargo", cargo.Export, cargo.Import},
		{"uv", uv.Export, uv.Import},
	} {
		exportCmd.AddCommand(newToolTransferCommand(tool.name, "export", tool.export))
		importCmd.AddCommand(newToolTransferCommand(tool.name, "import", tool.importTools))
	}
}

func newToolTransferCommand(name, direction string, action func(transfer.Options) error) *cobra.Command {
	var opts transfer.Options
	var apply bool
	title := "Export"
	dirHelp := "Output directory for exported files"
	if direction == "import" {
		title = "Import"
		dirHelp = "Directory containing exported files"
	}
	cmd := &cobra.Command{
		Use:     name,
		Short:   fmt.Sprintf("%s %s tools with pinned versions", title, name),
		Example: fmt.Sprintf("  goodbye %s %s --dir ~/goodbye-export\n  goodbye %s %s --dir ~/goodbye-export --apply", direction, name, direction, name),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DryRun = !apply
			opts.Out = cmd.OutOrStdout()
			return action(opts)
		},
	}
	cmd.Flags().StringVar(&opts.Dir, "dir", ".", dirHelp)
	cmd.Flags().BoolVar(&apply, "apply", false, "Actually perform the operation (default is dry-run)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Show installation output")
	if direction == "import" {
		cmd.Flags().BoolVar(&opts.Continue, "continue", false, "Continue installing after errors; still exit with an error if any fail")
	} else {
		cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format (text or mise; mise merges into .mise.toml)")
	}
	return cmd
}
