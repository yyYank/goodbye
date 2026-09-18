package uninstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var packageName = regexp.MustCompile(`^(@[A-Za-z0-9_~-][A-Za-z0-9._~-]*/)?[A-Za-z0-9_~][A-Za-z0-9._~-]*$`)
var toolLine = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*) v([^ :]+)(?: .*)?:?$`)

func List(manager string, run Runner) ([]Item, error) {
	var items []Item
	switch manager {
	case "go":
		return listGo(run)
	case "npm", "pnpm":
		root, err := run(manager, "root", "-g")
		if err != nil {
			return nil, err
		}
		dir := strings.TrimSpace(string(root))
		if !filepath.IsAbs(dir) {
			return nil, fmt.Errorf("invalid global root %q", dir)
		}
		if err := protectMise(dir); err != nil {
			return nil, err
		}
		data, err := run(manager, "list", "-g", "--depth=0", "--json")
		if err != nil {
			return nil, err
		}
		type dep struct{ Version string }
		type project struct {
			Dependencies         map[string]dep
			OptionalDependencies map[string]dep
			DevDependencies      map[string]dep
		}
		var projects []project
		if manager == "npm" {
			var p *project
			if err = json.Unmarshal(data, &p); err != nil {
				return nil, err
			}
			if p == nil {
				return nil, fmt.Errorf("invalid npm listing")
			}
			projects = []project{*p}
		} else {
			if err = json.Unmarshal(data, &projects); err != nil {
				return nil, err
			}
			if projects == nil {
				return nil, fmt.Errorf("invalid pnpm listing")
			}
		}
		seen := map[string]bool{}
		for _, p := range projects {
			for _, deps := range []map[string]dep{p.Dependencies, p.OptionalDependencies, p.DevDependencies} {
				for name, d := range deps {
					if !packageName.MatchString(name) {
						return nil, fmt.Errorf("invalid package name %q", name)
					}
					if seen[name] {
						continue
					}
					seen[name] = true
					action := "uninstall"
					if manager == "pnpm" {
						action = "remove"
					}
					items = append(items, Item{Name: name, Version: d.Version, Path: filepath.Join(dir, name), Command: []string{manager, action, "-g", "--", name}})
				}
			}
		}
	case "cargo", "uv":
		args := []string{"install", "--list", "--color", "never"}
		dir := ""
		if manager == "cargo" {
			var err error
			dir, err = cargoRoot()
			if err != nil {
				return nil, err
			}
			args = append(args, "--root", dir)
		}
		if manager == "uv" {
			args = []string{"tool", "list", "--color", "never"}
			data, err := run("uv", "tool", "dir")
			if err != nil {
				return nil, err
			}
			dir = strings.TrimSpace(string(data))
			if !filepath.IsAbs(dir) {
				return nil, fmt.Errorf("invalid uv tool directory")
			}
			if err := protectMise(dir); err != nil {
				return nil, err
			}
		}
		data, err := run(manager, args...)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || (manager == "uv" && strings.HasPrefix(line, "- ")) {
				continue
			}
			match := toolLine.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("unrecognized %s list entry %q", manager, line)
			}
			version := strings.TrimSuffix(match[2], ":")
			command := []string{"cargo", "uninstall", "--root", dir, "--", match[1]}
			if manager == "uv" {
				command = []string{"uv", "tool", "uninstall", "--", match[1]}
			}
			items = append(items, Item{Name: match[1], Version: version, Path: dir, Command: command})
		}
	case "brew":
		prefix, err := run("brew", "--prefix")
		if err != nil {
			return nil, err
		}
		root := strings.TrimSpace(string(prefix))
		if !filepath.IsAbs(root) {
			return nil, fmt.Errorf("invalid Homebrew prefix")
		}
		data, err := run("brew", "info", "--json=v2", "--installed")
		if err != nil {
			return nil, err
		}
		var listing *struct {
			Formulae []struct {
				Name      string
				FullName  string `json:"full_name"`
				Installed []struct{ Version string }
			}
			Casks []struct {
				Token     string
				Installed json.RawMessage
			}
		}
		if err = json.Unmarshal(data, &listing); err != nil {
			return nil, err
		}
		if listing == nil {
			return nil, fmt.Errorf("invalid brew listing")
		}
		for _, f := range listing.Formulae {
			name := f.FullName
			if name == "" {
				name = f.Name
			}
			var versions []string
			for _, v := range f.Installed {
				versions = append(versions, v.Version)
			}
			items = append(items, Item{Name: name, Version: strings.Join(versions, ","), Kind: "formula", Path: filepath.Join(root, "Cellar", f.Name), Command: []string{"brew", "uninstall", "--formula", "--", name}})
		}
		for _, c := range listing.Casks {
			var version string
			_ = json.Unmarshal(c.Installed, &version)
			items = append(items, Item{Name: c.Token, Version: version, Kind: "cask", Path: filepath.Join(root, "Caskroom", c.Token), Command: []string{"brew", "uninstall", "--cask", "--", c.Token}})
		}
	default:
		return nil, fmt.Errorf("unsupported manager %q", manager)
	}
	sortItems(items)
	return items, nil
}

// Reject isolated mise tool installations while allowing the language runtime's
// global bin/package directory (the source of native global installations).
func protectMise(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	normalized := filepath.ToSlash(filepath.Clean(path))
	markers := []string{"/mise/installs/"}
	if data := os.Getenv("MISE_DATA_DIR"); data != "" {
		markers = append(markers, filepath.ToSlash(filepath.Join(data, "installs"))+"/")
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			rest := strings.SplitN(normalized, marker, 2)[1]
			runtime := strings.Split(rest, "/")[0]
			if runtime != "go" && runtime != "node" && runtime != "rust" && runtime != "python" {
				return fmt.Errorf("refusing mise-managed tool directory: %s", path)
			}
		}
	}
	return nil
}
