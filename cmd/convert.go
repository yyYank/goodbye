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
	var to, from string
	c := &cobra.Command{
		Use:   "convert [command]",
		Short: "Convert an install command to a mise command without executing it",
		Long: `Convert one installation command supplied as a quoted argument, or multiple
commands on stdin. The conversion target defaults to mise. Use --from to read goodbye export package lists instead.
Blank lines and full-line comments (including shebangs) on stdin are ignored.
Supports go install, npm install/i -g, pnpm add -g, cargo install
(optionally --version), and uv tool install. Unsupported options and
shell expressions are rejected. Only the converted command is written to stdout.`,
		Example: `  echo 'go install github.com/foo/bar@latest' | goodbye convert
  goodbye convert 'npm install -g prettier'`,
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
			if len(args) == 1 && strings.ContainsAny(input, "\r\n") {
				return fmt.Errorf("use stdin for multiple lines")
			}
			result, err := convertLines(input, from)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result)
			return err
		},
	}
	c.Flags().StringVar(&to, "to", "mise", "Conversion target (mise)")
	c.Flags().StringVar(&from, "from", "", "Export source: go, npm, pnpm, cargo, uv (default: install commands)")
	return c
}

var (
	convertGoSpec       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*@[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	convertNpmSpec      = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*(@[A-Za-z0-9][A-Za-z0-9._+-]*)?$`)
	convertCrate        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
	convertCargoVersion = regexp.MustCompile(`^=?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?(\+[A-Za-z0-9.-]+)?$`)
	convertPythonSpec   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(==([0-9]+!)?[0-9][A-Za-z0-9._+-]*)?$`)
)

func convertLines(input, from string) (string, error) {
	prefixes := map[string]string{"go": "go install ", "npm": "npm install -g ", "pnpm": "pnpm add -g ", "cargo": "cargo install ", "uv": "uv tool install "}
	if from != "" && prefixes[from] == "" {
		return "", fmt.Errorf("unsupported source %q: expected go, npm, pnpm, cargo or uv", from)
	}
	var results []string
	for i, line := range strings.Split(input, "\n") {
		line = strings.Trim(strings.TrimSuffix(line, "\r"), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if from != "" {
			if strings.ContainsAny(line, " \t\r\"'") {
				return "", fmt.Errorf("line %d: expected one unquoted exported package", i+1)
			}
			if from == "cargo" {
				name, version, ok := strings.Cut(line, "@")
				if !ok || !convertCargoVersion.MatchString(version) {
					return "", fmt.Errorf("line %d: expected crate@version", i+1)
				}
				line = name + " --version " + version
			}
			line = prefixes[from] + line
		}
		result, err := convertInstallCommand(line)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", i+1, err)
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no install commands or packages found")
	}
	return strings.Join(results, "\n"), nil
}

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
	tool := backend + ":" + spec
	if strings.Contains(tool, "!") {
		tool = "'" + tool + "'"
	}
	return "mise use -g " + tool, nil
}
