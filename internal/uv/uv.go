package uv

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yyYank/goodbye/internal/transfer"
)

// uv displays normalized PEP 440 versions, including epochs and local versions.
var pinnedTool = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?==(?:[0-9]+!)?[0-9]+(?:\.[0-9]+)*(?:(?:a|b|rc)[0-9]+)?(?:\.post[0-9]+)?(?:\.dev[0-9]+)?(?:\+[a-z0-9]+(?:[._-][a-z0-9]+)*)?$`)
var registryRequirement = regexp.MustCompile(`^ \[required: [<>=!~0-9A-Za-z.*+, -]+\]$`)

func installArgs(spec string) ([]string, error) {
	if !pinnedTool.MatchString(spec) {
		return nil, fmt.Errorf("invalid pinned uv tool: %q", spec)
	}
	return []string{"tool", "install", "--", spec}, nil
}

func parseList(data []byte) ([]string, []string, error) {
	var items, warnings []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "- ") {
			continue
		}
		name, rest, ok := strings.Cut(line, " v")
		if !ok {
			return nil, nil, fmt.Errorf("unrecognized uv list entry: %q", line)
		}
		version, detail, hasDetail := strings.Cut(rest, " ")
		spec := name + "==" + version
		if _, err := installArgs(spec); err != nil {
			return nil, nil, err
		}
		if hasDetail && !registryRequirement.MatchString(" "+detail) {
			warnings = append(warnings, fmt.Sprintf("%s: source, extras or additional requirements cannot be restored (%s)", name, detail))
			continue
		}
		items = append(items, spec)
	}
	return items, warnings, scanner.Err()
}

func newManager(read, run transfer.Runner) transfer.Manager {
	return transfer.Manager{Name: "uv", File: "uv-tools.txt", InstallArgs: installArgs, Run: run, List: func() ([]string, []string, error) {
		data, err := read("uv", "tool", "list", "--show-version-specifiers", "--show-extras", "--show-with", "--color", "never", "--offline")
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
