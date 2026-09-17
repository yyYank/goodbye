// Package transfer handles the file and execution lifecycle of tool migrations.
package transfer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yyYank/goodbye/internal/miseconfig"
)

type Options struct {
	Dir                       string
	Format                    string
	DryRun, Verbose, Continue bool
	Out                       io.Writer
}

type Runner func(string, ...string) ([]byte, error)

// Output keeps command diagnostics separate from machine-readable stdout.
func Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return data, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return data, nil
}

func Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

type Manager struct {
	Name, File  string
	List        func() ([]string, []string, error)
	InstallArgs func(string) ([]string, error)
	Run         Runner
}

func (o Options) writer() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

func (o Options) directory() (string, error) {
	if o.Dir == "" {
		return ".", nil
	}
	if o.Dir == "~" || strings.HasPrefix(o.Dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(o.Dir, "~")), nil
	}
	return o.Dir, nil
}

func (m Manager) Export(opts Options) error {
	if opts.Format != "" && opts.Format != "text" && opts.Format != "mise" {
		return fmt.Errorf("invalid export format %q (must be text or mise)", opts.Format)
	}
	dir, err := opts.directory()
	if err != nil {
		return err
	}
	items, warnings, err := m.List()
	if err != nil {
		return fmt.Errorf("list %s tools: %w", m.Name, err)
	}
	out := opts.writer()
	for _, warning := range warnings {
		fmt.Fprintf(out, "Warning: skipped %s\n", warning)
	}
	sort.Strings(items)
	var lines []string
	for _, item := range items {
		if _, err := m.InstallArgs(item); err != nil {
			return err
		}
		if len(lines) == 0 || lines[len(lines)-1] != item {
			lines = append(lines, item)
		}
	}
	if opts.Format == "mise" {
		tools, err := miseVersions(m.Name, lines)
		if err != nil {
			return err
		}
		return miseconfig.Export(dir, tools, opts.DryRun, out)
	}
	path := filepath.Join(dir, m.File)
	if opts.DryRun {
		fmt.Fprintf(out, "[dry-run] Would export %d tools to %s\n", len(lines), path)
		for _, line := range lines {
			fmt.Fprintln(out, "  "+line)
		}
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Fprintf(out, "Exported %d tools to %s\n", len(lines), path)
	return nil
}

func (m Manager) Import(opts Options) error {
	dir, err := opts.directory()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, m.File)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var specs []string
	var commands [][]string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		spec := strings.TrimSpace(scanner.Text())
		if spec == "" || strings.HasPrefix(spec, "#") {
			continue
		}
		args, err := m.InstallArgs(spec)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if seen[spec] {
			continue
		}
		seen[spec] = true
		specs = append(specs, spec)
		commands = append(commands, args)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	out := opts.writer()
	run := m.Run
	if run == nil {
		run = Run
	}
	var failures []error
	succeeded := 0
	for i, args := range commands {
		if opts.DryRun {
			fmt.Fprintf(out, "[dry-run] %s %s\n", m.Name, strings.Join(args, " "))
			continue
		}
		fmt.Fprintf(out, "Installing %s...\n", specs[i])
		data, err := run(m.Name, args...)
		if opts.Verbose || err != nil {
			fmt.Fprint(out, string(data))
		}
		if err != nil {
			failure := fmt.Errorf("install %s: %w", specs[i], err)
			fmt.Fprintln(out, failure)
			failures = append(failures, failure)
			if !opts.Continue {
				break
			}
			continue
		}
		succeeded++
	}
	if !opts.DryRun {
		fmt.Fprintf(out, "Import summary: %d succeeded, %d failed\n", succeeded, len(failures))
	}
	return errors.Join(failures...)
}
