package npm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yyYank/goodbye/internal/transfer"
)

var pinnedPackage = regexp.MustCompile(`^(?:@[a-zA-Z0-9_~-][a-zA-Z0-9._~-]*/)?[a-zA-Z0-9_~][a-zA-Z0-9._~-]*@[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func validatePinnedPackage(spec string) ([]string, error) {
	if !pinnedPackage.MatchString(spec) {
		return nil, fmt.Errorf("invalid pinned npm package: %q", spec)
	}
	return nil, nil
}

func parsePinnedPackages(data []byte) ([]string, []string, error) {
	var listing *struct {
		Dependencies map[string]struct {
			Name, Version, Resolved string
			Link                    bool
		}
	}
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, nil, fmt.Errorf("parse npm list: %w", err)
	}
	if listing == nil {
		return nil, nil, fmt.Errorf("expected npm list JSON object")
	}
	var items, warnings []string
	for name, pkg := range listing.Dependencies {
		if pkg.Link || (pkg.Name != "" && pkg.Name != name) || strings.HasPrefix(pkg.Resolved, "git") || strings.HasPrefix(pkg.Resolved, "file:") {
			warnings = append(warnings, fmt.Sprintf("%s: local link, alias or non-registry source", name))
			continue
		}
		spec := name + "@" + pkg.Version
		if _, err := validatePinnedPackage(spec); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: missing or unsupported version %q", name, pkg.Version))
			continue
		}
		items = append(items, spec)
	}
	sort.Strings(items)
	sort.Strings(warnings)
	return items, warnings, nil
}

func exportMise(opts ExportOptions) error {
	manager := transfer.Manager{Name: "npm", InstallArgs: validatePinnedPackage, List: func() ([]string, []string, error) {
		data, err := transfer.Output("npm", "list", "-g", "--depth=0", "--json", "--long")
		if err != nil {
			return nil, nil, err
		}
		return parsePinnedPackages(data)
	}}
	return manager.Export(transfer.Options{Dir: opts.Dir, Format: "mise", DryRun: opts.DryRun, Verbose: opts.Verbose, Out: opts.Out})
}
