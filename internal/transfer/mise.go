package transfer

import (
	"fmt"
	"strings"
)

func miseVersions(source string, specs []string) (map[string]string, error) {
	backend := map[string]string{"npm": "npm", "pnpm": "npm", "go": "go", "cargo": "cargo", "uv": "pypi"}[source]
	if backend == "" {
		return nil, fmt.Errorf("mise export is not supported for %s", source)
	}
	tools := make(map[string]string)
	for _, spec := range specs {
		separator := "@"
		if source == "uv" {
			separator = "=="
		}
		index := strings.LastIndex(spec, separator)
		if index <= 0 || index+len(separator) == len(spec) {
			return nil, fmt.Errorf("missing pinned version: %q", spec)
		}
		key, version := backend+":"+spec[:index], spec[index+len(separator):]
		if previous, ok := tools[key]; ok && previous != version {
			return nil, fmt.Errorf("conflict for %s: exported versions %s and %s", key, previous, version)
		}
		tools[key] = version
	}
	return tools, nil
}
