// Package miseconfig merges exported tool versions into a mise configuration.
package miseconfig

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

// Export keeps existing values and rejects conflicting tool definitions.
// When adding entries, TOML is re-encoded; comments and formatting are not retained.
func Export(dir string, additions map[string]string, dryRun bool, out io.Writer) error {
	path := filepath.Join(dir, ".mise.toml")
	if !dryRun {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		lock, err := os.OpenFile(path+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("lock %s (another export may be running): %w", path, err)
		}
		defer os.Remove(lock.Name())
		if err := lock.Close(); err != nil {
			return err
		}
	}
	data, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	config := make(map[string]any)
	if _, err := toml.Decode(string(data), &config); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	tools := make(map[string]any)
	if value, ok := config["tools"]; ok {
		var valid bool
		tools, valid = value.(map[string]any)
		if !valid {
			return fmt.Errorf("%s: tools must be a TOML table", path)
		}
	}
	keys := make([]string, 0, len(additions))
	for key := range additions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	changed := !exists
	for _, key := range keys {
		version := additions[key]
		if previous, ok := tools[key]; ok {
			if previous != version {
				return fmt.Errorf("%s: conflict for %s: existing %v, exported %s", path, key, previous, version)
			}
			continue
		}
		tools[key] = version
		changed = true
	}
	config["tools"] = tools
	var content bytes.Buffer
	if err := toml.NewEncoder(&content).Encode(config); err != nil {
		return err
	}
	if dryRun {
		fmt.Fprintf(out, "[dry-run] Would merge %d tools into %s\n%s", len(additions), path, content.String())
		return nil
	}
	if !changed {
		fmt.Fprintf(out, "No changes to %s\n", path)
		return nil
	}
	mode := os.FileMode(0644)
	if exists {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode = info.Mode().Perm()
	}
	file, err := os.CreateTemp(dir, ".mise-export-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := file.Write(content.Bytes()); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Merged %d tools into %s\n", len(additions), path)
	return nil
}
