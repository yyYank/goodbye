package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(newConvertCommand()) }

func newConvertCommand() *cobra.Command {
	var to string
	c := &cobra.Command{
		Use:   "convert --to mise [command]",
		Short: "Convert an install command to a mise command without executing it",
		Long: `Convert one installation command supplied as a quoted argument or on stdin.
Supports go install, npm install/i -g, pnpm add -g, cargo install
(optionally --version), and uv tool install. Unsupported options and
shell expressions are rejected. Only the converted command is written to stdout.`,
		Example: `  echo 'go install github.com/foo/bar@latest' | goodbye convert --to mise
  goodbye convert --to mise 'npm install -g prettier'`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if to != "mise" {
				return fmt.Errorf("unsupported conversion target %q: use --to mise", to)
			}
			var input string
			if len(args) == 1 {
				input = args[0]
			} else {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("read install command: %w", err)
				}
				input = string(data)
			}
			result, err := convertInstallCommand(input)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result)
			return err
		},
	}
	c.Flags().StringVar(&to, "to", "", "Conversion target (mise)")
	return c
}

var (
	convertGoSpec       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*@[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	convertNpmSpec      = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*(@[A-Za-z0-9][A-Za-z0-9._+-]*)?$`)
	convertCrate        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
	convertCargoVersion = regexp.MustCompile(`^=?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?(\+[A-Za-z0-9.-]+)?$`)
	convertPythonSpec   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(==[0-9][A-Za-z0-9._+]*)?$`)
)

func convertInstallCommand(input string) (string, error) {
	// This deliberately accepts a small literal grammar, not a shell program.
	// All emitted characters are shell-safe; expansions are never evaluated.
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, "\r")
	if strings.ContainsAny(input, "\n\r") {
		return "", fmt.Errorf("expected one install command; multiple lines are unsupported")
	}
	words := strings.FieldsFunc(input, func(r rune) bool { return r == ' ' || r == '\t' })
	for i, word := range words {
		if len(word) >= 2 && (word[0] == '\'' || word[0] == '"') && word[len(word)-1] == word[0] {
			words[i] = word[1 : len(word)-1]
		}
	}
	invalid := func() (string, error) {
		return "", fmt.Errorf("unsupported or ambiguous install command: use a supported command with one literal package and no unsupported options")
	}
	if len(words) < 3 {
		return invalid()
	}
	var backend, spec string
	switch {
	case words[0] == "go" && words[1] == "install" && len(words) == 3:
		backend, spec = "go", words[2]
		if !convertGoSpec.MatchString(spec) || strings.Contains(spec, "...") {
			return invalid()
		}
	case ((words[0] == "npm" && (words[1] == "install" || words[1] == "i")) || (words[0] == "pnpm" && words[1] == "add")) && len(words) == 4 && words[2] == "-g":
		backend, spec = "npm", words[3]
		if !convertNpmSpec.MatchString(spec) {
			return invalid()
		}
	case words[0] == "cargo" && words[1] == "install":
		backend, spec = "cargo", words[2]
		if !convertCrate.MatchString(spec) {
			return invalid()
		}
		if len(words) == 5 && words[3] == "--version" && convertCargoVersion.MatchString(words[4]) {
			spec += "@" + strings.TrimPrefix(words[4], "=")
		} else if len(words) != 3 {
			return invalid()
		}
	case words[0] == "uv" && words[1] == "tool" && words[2] == "install" && len(words) == 4:
		backend, spec = "pypi", words[3]
		if !convertPythonSpec.MatchString(spec) {
			return invalid()
		}
		spec = strings.Replace(spec, "==", "@", 1)
	default:
		return invalid()
	}
	return "mise use -g " + backend + ":" + spec, nil
}
