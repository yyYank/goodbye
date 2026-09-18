package uninstall

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

func listGo(run Runner) ([]Item, error) {
	data, err := run("go", "env", "-json", "GOBIN", "GOPATH")
	if err != nil {
		return nil, err
	}
	var env struct{ GOBIN, GOPATH string }
	if err = json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	dir := env.GOBIN
	if dir == "" {
		paths := filepath.SplitList(env.GOPATH)
		if len(paths) == 0 || paths[0] == "" {
			return nil, fmt.Errorf("missing Go bin directory")
		}
		dir = filepath.Join(paths[0], "bin")
	}
	return scanGo(dir, buildinfo.ReadFile)
}

func scanGo(dir string, read func(string) (*debug.BuildInfo, error)) ([]Item, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("Go bin directory must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dir = resolved
	if err = protectMise(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var items []Item
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		stat, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !stat.Mode().IsRegular() || stat.Mode()&0111 == 0 {
			continue
		}
		info, err := read(path)
		if err != nil || info.Main.Path == "" || info.Path == "" || (info.Path != info.Main.Path && !strings.HasPrefix(info.Path, info.Main.Path+"/")) {
			continue
		}
		check := func() error {
			parent, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return err
			}
			if parent != dir {
				return fmt.Errorf("Go bin directory changed: %s", dir)
			}
			current, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if !current.Mode().IsRegular() || !os.SameFile(stat, current) || stat.Size() != current.Size() || !stat.ModTime().Equal(current.ModTime()) {
				return fmt.Errorf("Go binary changed since selection: %s", path)
			}
			return nil
		}
		items = append(items, Item{Name: info.Path, Version: info.Main.Version, Path: path, Kind: "go", check: check, remove: func() error {
			if err := check(); err != nil {
				return err
			}
			return os.Remove(path)
		}})
	}
	sortItems(items)
	return items, nil
}
