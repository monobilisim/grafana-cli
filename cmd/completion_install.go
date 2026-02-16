package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var completionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Automatically install shell completion for gcli",
	Long: `Detects your current shell (bash or zsh) and adds the necessary 
sourcing line to your shell profile (~/.bashrc or ~/.zshrc) to enable 
gcli auto-completion.`,
	Run: func(cmd *cobra.Command, args []string) {
		shell := detectShell()
		if shell == "" {
			fmt.Println("Could not detect shell. Please install completion manually.")
			return
		}

		err := installForShell(shell)
		if err != nil {
			fmt.Printf("Error installing completion: %v\n", err)
			return
		}

		fmt.Printf("Successfully installed gcli completion for %s. Please restart your shell or run 'source ~/.%src'\n", shell, shell)
	},
}

var completionUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Automatically remove gcli shell completion",
	Long: `Detects your current shell (bash or zsh) and removes the 
gcli sourcing line from your shell profile (~/.bashrc or ~/.zshrc).`,
	Run: func(cmd *cobra.Command, args []string) {
		shell := detectShell()
		if shell == "" {
			fmt.Println("Could not detect shell. Please uninstall completion manually.")
			return
		}

		err := uninstallForShell(shell)
		if err != nil {
			fmt.Printf("Error uninstalling completion: %v\n", err)
			return
		}

		fmt.Printf("Successfully uninstalled gcli completion for %s. Please restart your shell.\n", shell)
	},
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if strings.Contains(shellPath, "zsh") {
		return "zsh"
	}
	if strings.Contains(shellPath, "bash") {
		return "bash"
	}
	// Fallback to checking the parent process if SHELL is not set or not helpful
	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", os.Getppid()), "-o", "comm=").Output()
	if err == nil {
		comm := strings.TrimSpace(string(out))
		if strings.Contains(comm, "zsh") {
			return "zsh"
		}
		if strings.Contains(comm, "bash") {
			return "bash"
		}
	}
	return ""
}

func installForShell(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	rcFile := filepath.Join(home, "."+shell+"rc")

	// Create completion file directory
	compDir := filepath.Join(home, ".gcli", "completion")
	if err := os.MkdirAll(compDir, 0755); err != nil {
		return err
	}

	compFile := filepath.Join(compDir, "gcli."+shell)

	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}

	// Generate completion script
	var genCmd *exec.Cmd
	if shell == "zsh" {
		genCmd = exec.Command(self, "completion", "zsh")
	} else {
		genCmd = exec.Command(self, "completion", "bash")
	}

	out, err := genCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate completion script: %w", err)
	}

	if err := os.WriteFile(compFile, out, 0644); err != nil {
		return err
	}

	// Add sourcing line to rc file if not exists
	sourceLine := fmt.Sprintf("source %s", compFile)
	if shell == "zsh" {
		// For zsh, sometimes we need to ensure compinit is called,
		// but usually gcli completion zsh handles its own requirements.
		sourceLine = fmt.Sprintf("[[ -f %s ]] && source %s", compFile, compFile)
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(rcFile, []byte(sourceLine+"\n"), 0644)
		}
		return err
	}

	if !strings.Contains(string(content), compFile) {
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString("\n# gcli completion\n" + sourceLine + "\n")
		return err
	}

	return nil
}

func uninstallForShell(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	rcFile := filepath.Join(home, "."+shell+"rc")
	compDir := filepath.Join(home, ".gcli", "completion")
	compFile := filepath.Join(compDir, "gcli."+shell)

	// Remove completion script
	_ = os.Remove(compFile)

	// Remove sourcing line from rc file
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, "# gcli completion") {
			skip = true
			continue
		}
		if skip && (strings.Contains(line, compFile) || strings.Contains(line, "source "+compFile)) {
			skip = false
			continue
		}
		newLines = append(newLines, line)
	}

	return os.WriteFile(rcFile, []byte(strings.Join(newLines, "\n")), 0644)
}

func init() {
	var completionCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "completion" {
			completionCmd = c
			break
		}
	}

	if completionCmd == nil {
		completionCmd = &cobra.Command{
			Use:   "completion",
			Short: "Generate autocompletion script and install it",
			Long: `Generate the autocompletion script for the specified shell.
See each sub-command's help for details on how to use the generated script.`,
		}

		bash := &cobra.Command{
			Use:   "bash",
			Short: "Generate the autocompletion script for bash",
			Run: func(cmd *cobra.Command, args []string) {
				rootCmd.GenBashCompletion(os.Stdout)
			},
		}

		zsh := &cobra.Command{
			Use:   "zsh",
			Short: "Generate the autocompletion script for zsh",
			Run: func(cmd *cobra.Command, args []string) {
				rootCmd.GenZshCompletion(os.Stdout)
			},
		}

		fish := &cobra.Command{
			Use:   "fish",
			Short: "Generate the autocompletion script for fish",
			Run: func(cmd *cobra.Command, args []string) {
				rootCmd.GenFishCompletion(os.Stdout, true)
			},
		}

		powershell := &cobra.Command{
			Use:   "powershell",
			Short: "Generate the autocompletion script for powershell",
			Run: func(cmd *cobra.Command, args []string) {
				rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			},
		}

		completionCmd.AddCommand(bash, zsh, fish, powershell)
		rootCmd.AddCommand(completionCmd)
	}

	completionCmd.AddCommand(completionInstallCmd, completionUninstallCmd)
	rootCmd.AddCommand(completionCmd)
}
