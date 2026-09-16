package pnpm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Export(opts transfer.Options) error {
	return newManager(transfer.Output, transfer.Run).Export(opts)
}
func Import(opts transfer.Options) error {
	return newManager(transfer.Output, transfer.Run).Import(opts)
}

func newManager(read, run transfer.Runner) transfer.Manager {
	return transfer.Manager{Name: "pnpm", File: "pnpm-global.txt", InstallArgs: installArgs, Run: run, List: func() ([]string, []string, error) {
		data, err := read("pnpm", "list", "-g", "--depth=0", "--json")
		if err != nil {
			return nil, nil, err
		}
		return parseList(data)
	}}
}

var packageSpec = regexp.MustCompile(`^(?:@[a-zA-Z0-9_~-][a-zA-Z0-9._~-]*/)?[a-zA-Z0-9_~][a-zA-Z0-9._~-]*@[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func installArgs(spec string) ([]string, error) {
	if !packageSpec.MatchString(spec) {
		return nil, fmt.Errorf("invalid pinned pnpm package: %q", spec)
	}
	return []string{"add", "-g", "--", spec}, nil
}

func parseList(data []byte) ([]string, []string, error) {
	var projects []struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, nil, fmt.Errorf("parse pnpm list: %w", err)
	}
	if projects == nil {
		return nil, nil, fmt.Errorf("expected pnpm list JSON array")
	}
	var items, warnings []string
	for _, project := range projects {
		for name, pkg := range project.Dependencies {
			spec := name + "@" + pkg.Version
			if _, err := installArgs(spec); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: unsupported source or version %q", name, pkg.Version))
				continue
			}
			items = append(items, spec)
		}
	}
	sort.Strings(items)
	sort.Strings(warnings)
	return items, warnings, nil
}
