package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/transfer"
	"github.com/yyYank/goodbye/internal/uninstall"
)

func init() {
	run := uninstall.Runner(transfer.Output)
	rootCmd.AddCommand(newUninstallCommand(func(manager string) ([]uninstall.Item, error) { return uninstall.List(manager, run) }, run))
}

func newUninstallCommand(list func(string) ([]uninstall.Item, error), run uninstall.Runner) *cobra.Command {
	parent := &cobra.Command{Use: "uninstall", Short: "Remove native global packages (dry-run by default)", Args: cobra.NoArgs, SilenceUsage: true}
	for _, manager := range []string{"npm", "pnpm", "go", "cargo", "uv", "brew"} {
		var all, apply, tui bool
		var file string
		c := &cobra.Command{
			Use: manager + " [packages...]", Short: "Uninstall from " + manager,
			Long: "Select installed packages by name, --all, --file, or --tui.\nDefaults to dry-run. --apply performs deletion. Interactive deletion also requires typing delete.\nOnly the selected manager's current global environment is targeted.",
			RunE: func(cmd *cobra.Command, args []string) error {
				modes := 0
				if len(args) > 0 {
					modes++
				}
				if all {
					modes++
				}
				if file != "" {
					modes++
				}
				if tui {
					modes++
				}
				if modes != 1 {
					return fmt.Errorf("choose exactly one of package names, --all, --file or --tui")
				}
				if file != "" {
					data, err := os.ReadFile(file)
					if err != nil {
						return err
					}
					for _, line := range strings.Split(string(data), "\n") {
						line = strings.TrimSpace(line)
						if line != "" && !strings.HasPrefix(line, "#") {
							args = append(args, line)
						}
					}
					if len(args) == 0 {
						return fmt.Errorf("no packages in %s", file)
					}
				}
				items, err := list(manager)
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				if len(items) == 0 && (all || tui) {
					_, err = fmt.Fprintln(out, "No installed packages found.")
					return err
				}
				var selected []uninstall.Item
				if tui {
					reader := bufio.NewReader(cmd.InOrStdin())
					selected, err = selectUninstallTUI(items, reader, cmd.InOrStdin(), out)
					if err != nil {
						return err
					}
					if len(selected) == 0 {
						return nil
					}
					if err = uninstall.Execute(selected, false, out, run); err != nil {
						return err
					}
					if !apply {
						return nil
					}
					fmt.Fprint(out, "Type delete to remove these packages (anything else cancels): ")
					answer, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("read deletion confirmation: %w", err)
					}
					if strings.TrimSpace(answer) != "delete" {
						fmt.Fprintln(out, "Cancelled.")
						return nil
					}
				} else {
					selected, err = uninstall.Select(items, args, all)
					if err != nil {
						return err
					}
				}
				return uninstall.Execute(selected, apply, out, run)
			},
		}
		c.Flags().BoolVar(&all, "all", false, "Select all installed packages in this manager's global environment")
		c.Flags().BoolVar(&apply, "apply", false, "Actually uninstall (default is dry-run)")
		c.Flags().BoolVar(&tui, "tui", false, "Select packages with arrow keys/hjkl and Space")
		c.Flags().StringVar(&file, "file", "", "Read package names or pinned specs from an export text file")
		parent.AddCommand(c)
	}
	return parent
}
