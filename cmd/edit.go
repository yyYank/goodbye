package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the goodbye.toml configuration file",
	Long: `Open the ~/.goodbye.toml configuration file in your preferred editor.

The editor can be specified with the --editor flag. If not specified,
it uses the EDITOR environment variable, or falls back to vim.`,
	Example: `  # Open with default editor (EDITOR env var or vim)
  goodbye edit

  # Open with a specific editor
  goodbye edit --editor vim
  goodbye edit --editor emacs
  goodbye edit --editor nano
  goodbye edit --editor code`,
	RunE: runEdit,
}

var editEditor string

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVarP(&editEditor, "editor", "e", "", "Editor to use (default: $EDITOR or vim)")
}

func runEdit(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	editor := getEditor(editEditor)

	created, err := ensureConfigFileExists(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	if created {
		fmt.Printf("Config file does not exist. Creating %s...\n", configPath)
	}

	fmt.Printf("Opening %s with %s...\n", configPath, editor)

	// Execute the editor
	execCmd := exec.Command(editor, configPath)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the goodbye.toml config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".goodbye.toml"), nil
}

// getEditor determines which editor to use based on flag, env var, or default
func getEditor(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		return envEditor
	}
	return "vim"
}

// ensureConfigFileExists creates the config file if it doesn't exist
// Returns true if the file was created, false if it already existed
func ensureConfigFileExists(configPath string) (bool, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		file, err := os.Create(configPath)
		if err != nil {
			return false, err
		}
		file.Close()
		return true, nil
	}
	return false, nil
}
