// Package uninstall plans removals from explicitly selected package managers.
package uninstall

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type Runner func(string, ...string) ([]byte, error)
type Item struct {
	Name, Version, Path, Kind string
	Command                   []string
	check                     func() error
	remove                    func() error
}

func Select(items []Item, names []string, all bool) ([]Item, error) {
	if all {
		if len(names) > 0 {
			return nil, fmt.Errorf("--all cannot be combined with package names")
		}
		return items, nil
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("specify packages, --all, --file or --tui")
	}
	selected := map[int]bool{}
	for _, spec := range names {
		name, version := spec, ""
		exact := false
		for _, item := range items {
			if spec == item.Name || spec == item.Kind+":"+item.Name {
				exact = true
			}
		}
		if !exact {
			if i := strings.LastIndex(spec, "@"); i > 0 {
				name, version = spec[:i], spec[i+1:]
			}
			if i := strings.Index(spec, "=="); i > 0 {
				name, version = spec[:i], spec[i+2:]
			}
			if name != spec && version == "" {
				return nil, fmt.Errorf("empty version: %q", spec)
			}
		}
		count := 0
		for i, item := range items {
			if name != item.Name && name != item.Kind+":"+item.Name {
				continue
			}
			if version != "" && version != item.Version {
				continue
			}
			count++
			selected[i] = true
		}
		if count != 1 {
			return nil, fmt.Errorf("%q: expected one installed match, found %d (check name/version or use kind:name)", spec, count)
		}
	}
	var result []Item
	for i, item := range items {
		if selected[i] {
			result = append(result, item)
		}
	}
	return result, nil
}

func Execute(items []Item, apply bool, out io.Writer, run Runner) error {
	for _, item := range items {
		if item.check != nil {
			if err := item.check(); err != nil {
				return err
			}
		}
		if item.remove == nil && len(item.Command) < 2 {
			return fmt.Errorf("missing uninstall command for %s", item.Name)
		}
	}
	for _, item := range items {
		mode := "dry-run"
		if apply {
			mode = "delete"
		}
		if _, err := fmt.Fprintf(out, "[%s] %s %s %s\n", mode, item.Name, item.Version, item.Path); err != nil {
			return err
		}
	}
	if !apply {
		return nil
	}
	for i := 0; i < len(items); i++ {
		item := items[i]
		if item.remove != nil {
			if err := item.remove(); err != nil {
				return err
			}
			continue
		}
		command := append([]string(nil), item.Command...)
		if command[0] == "brew" {
			prefix := strings.Join(command[:len(command)-1], "\x00")
			for i+1 < len(items) {
				next := items[i+1].Command
				if len(next) < 2 || strings.Join(next[:len(next)-1], "\x00") != prefix {
					break
				}
				command = append(command, next[len(next)-1])
				i++
			}
		}
		data, err := run(command[0], command[1:]...)
		if len(data) > 0 {
			if _, writeErr := fmt.Fprint(out, string(data)); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			return fmt.Errorf("uninstall %s: %w", item.Name, err)
		}
	}
	return nil
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		// Keep the npm executable available until other npm packages are removed.
		if items[i].Name == "npm" {
			return false
		}
		if items[j].Name == "npm" {
			return true
		}
		return items[i].Kind+items[i].Name < items[j].Kind+items[j].Name
	})
}
