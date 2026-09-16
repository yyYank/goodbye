package cargo

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yyYank/goodbye/internal/transfer"
)

var pinnedCrate = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*@[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
var listEntry = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_-]*) v([^ ]+)(?: \((.+)\))?:$`)

func installArgs(spec string) ([]string, error) {
	if !pinnedCrate.MatchString(spec) {
		return nil, fmt.Errorf("invalid pinned Cargo crate: %q", spec)
	}
	name, version, _ := strings.Cut(spec, "@")
	return []string{"install", name, "--version", "=" + version}, nil
}

func parseList(data []byte) ([]string, []string, error) {
	var items, warnings []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		match := listEntry.FindStringSubmatch(line)
		if match == nil {
			return nil, nil, fmt.Errorf("unrecognized cargo list entry: %q", line)
		}
		if match[3] != "" {
			warnings = append(warnings, fmt.Sprintf("%s: non-default source %s", match[1], match[3]))
			continue
		}
		spec := match[1] + "@" + match[2]
		if _, err := installArgs(spec); err != nil {
			return nil, nil, err
		}
		items = append(items, spec)
	}
	return items, warnings, scanner.Err()
}

func newManager(read, run transfer.Runner) transfer.Manager {
	return transfer.Manager{Name: "cargo", File: "cargo-tools.txt", InstallArgs: installArgs, Run: run, List: func() ([]string, []string, error) {
		data, err := read("cargo", "install", "--list", "--color", "never")
		if err != nil {
			return nil, nil, err
		}
		return parseList(data)
	}}
}
func Export(opts transfer.Options) error {
	return newManager(transfer.Output, transfer.Run).Export(opts)
}
func Import(opts transfer.Options) error {
	return newManager(transfer.Output, transfer.Run).Import(opts)
}
