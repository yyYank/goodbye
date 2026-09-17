package gotools

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/yyYank/goodbye/internal/transfer"
)

var pinnedPath = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*\.[A-Za-z0-9.-]+(?:/[A-Za-z0-9_~.-]+)+@v[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?$`)

func installArgs(spec string) ([]string, error) {
	if !pinnedPath.MatchString(spec) {
		return nil, fmt.Errorf("invalid pinned Go package: %q", spec)
	}
	path, _, _ := strings.Cut(spec, "@")
	for _, part := range strings.Split(path, "/") {
		if part == "." || part == ".." || strings.Contains(part, "...") {
			return nil, fmt.Errorf("invalid Go package path: %q", path)
		}
	}
	return []string{"install", spec}, nil
}

func buildSpec(info *debug.BuildInfo) (string, error) {
	if strings.HasSuffix(info.Main.Version, "+dirty") {
		return "", fmt.Errorf("local changes cannot be restored from a module version")
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.modified" && setting.Value == "true" {
			return "", fmt.Errorf("local changes cannot be restored from a module version")
		}
	}
	if info.Main.Replace != nil {
		return "", fmt.Errorf("main module has a replacement")
	}
	for _, dep := range info.Deps {
		if dep.Replace != nil {
			return "", fmt.Errorf("dependency %s has a replacement", dep.Path)
		}
	}
	if info.Main.Path == "" || (info.Path != info.Main.Path && !strings.HasPrefix(info.Path, info.Main.Path+"/")) {
		return "", fmt.Errorf("command path is not in main module")
	}
	spec := info.Path + "@" + info.Main.Version
	if _, err := installArgs(spec); err != nil {
		return "", err
	}
	return spec, nil
}

func binDir(data []byte) (string, error) {
	var env struct{ GOBIN, GOPATH string }
	if err := json.Unmarshal(data, &env); err != nil {
		return "", fmt.Errorf("parse go env: %w", err)
	}
	dir := env.GOBIN
	if dir == "" {
		paths := filepath.SplitList(env.GOPATH)
		if len(paths) == 0 || paths[0] == "" {
			return "", fmt.Errorf("go env returned no GOBIN or GOPATH")
		}
		dir = filepath.Join(paths[0], "bin")
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("Go bin directory must be absolute: %q", dir)
	}
	return dir, nil
}

func scanDir(dir string, read func(string) (*debug.BuildInfo, error)) ([]string, []string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var items, warnings []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		stat, err := os.Stat(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if !stat.Mode().IsRegular() {
			continue
		}
		info, err := read(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		spec, err := buildSpec(info)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		items = append(items, spec)
	}
	return items, warnings, nil
}

func newManager(read, run transfer.Runner, scan func(string) ([]string, []string, error)) transfer.Manager {
	return transfer.Manager{Name: "go", File: "go-tools.txt", InstallArgs: installArgs, Run: run, List: func() ([]string, []string, error) {
		data, err := read("go", "env", "-json", "GOBIN", "GOPATH")
		if err != nil {
			return nil, nil, err
		}
		dir, err := binDir(data)
		if err != nil {
			return nil, nil, err
		}
		return scan(dir)
	}}
}

func manager() transfer.Manager {
	return newManager(transfer.Output, transfer.Run, func(dir string) ([]string, []string, error) { return scanDir(dir, buildinfo.ReadFile) })
}
func Export(opts transfer.Options) error { return manager().Export(opts) }
func Import(opts transfer.Options) error { return manager().Import(opts) }
