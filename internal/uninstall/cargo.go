package uninstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func cargoRoot() (string, error) {
	home := os.Getenv("CARGO_HOME")
	if home == "" {
		user, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(user, ".cargo")
	}
	root := os.Getenv("CARGO_INSTALL_ROOT")
	if root == "" {
		// Cargo install reads installation configuration from CARGO_HOME. The legacy
		// config filename takes precedence when both files exist.
		for _, name := range []string{"config", "config.toml"} {
			data, err := os.ReadFile(filepath.Join(home, name))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return "", err
			}
			var config struct{ Install struct{ Root string } }
			if _, err = toml.Decode(string(data), &config); err != nil {
				return "", fmt.Errorf("read Cargo install root: %w", err)
			}
			root = config.Install.Root
			if root != "" && !filepath.IsAbs(root) {
				return "", fmt.Errorf("relative Cargo install.root is unsupported; set CARGO_INSTALL_ROOT to an absolute path")
			}
			break
		}
	}
	if root == "" {
		root = home
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err = protectMise(root); err != nil {
		return "", err
	}
	return root, nil
}
