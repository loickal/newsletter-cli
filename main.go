package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/loickal/newsletter-cli/cmd"
	"golang.org/x/term"
)

var version = "0.3.0"

func main() {
	// Detect if running from GUI (double-click) vs CLI
	if isGUILaunch() {
		launchInTerminal()
		return
	}

	cmd.SetVersion(version)
	cmd.Execute()
}

func Version() string {
	return version
}

// isGUILaunch detects if the app was launched from GUI (double-click)
// Returns true if stdin is not a terminal
func isGUILaunch() bool {
	// Check if stdin is a terminal
	// If not, likely launched from GUI
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

// launchInTerminal opens the app in a terminal window
func launchInTerminal() {
	// Get the executable path
	exePath, err := os.Executable()
	if err != nil {
		// Fallback: try to find ourselves
		exePath = os.Args[0]
	}
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		absPath = exePath
	}

	switch runtime.GOOS {
	case "darwin":
		launchInTerminalMacOS(absPath)
	case "linux":
		launchInTerminalLinux(absPath)
	case "windows":
		launchInTerminalWindows(absPath)
	}
}

// launchInTerminalMacOS opens the app in macOS Terminal or iTerm
func launchInTerminalMacOS(absPath string) {
	script := `tell application "System Events"
	set terminalApp to ""
	
	-- Check if iTerm is installed
	try
		tell application "iTerm" to get version
		set terminalApp to "iTerm"
	on error
		-- Check if iTerm2 is installed
		try
			tell application "iTerm2" to get version
			set terminalApp to "iTerm2"
		on error
			-- Fallback to Terminal
			set terminalApp to "Terminal"
		end try
	end try
	
	if terminalApp is "iTerm" or terminalApp is "iTerm2" then
		tell application terminalApp
			activate
			tell current window
				tell current session
					write text "cd "$HOME" && clear && ` + absPath + `; echo; echo 'Press Enter to exit...'; read"
				end tell
			end tell
		end tell
	else
		tell application "Terminal"
			activate
			do script "cd "$HOME" && clear && ` + absPath + `; echo; echo 'Press Enter to exit...'; read"
		end tell
	end if
end tell`

	cmd := exec.Command("osascript", "-e", script)
	cmd.Run()
}

// launchInTerminalLinux opens the app in a Linux terminal emulator
func launchInTerminalLinux(absPath string) {
	// Try to find the best available terminal
	terminals := []struct {
		name string
		args []string
	}{
		{"gnome-terminal", []string{"--", "bash", "-c"}},
		{"konsole", []string{"-e", "bash", "-c"}},
		{"xterm", []string{"-e", "bash", "-c"}},
		{"x-terminal-emulator", []string{"-e", "bash", "-c"}},
		{"terminator", []string{"-e", "bash", "-c"}},
		{"mate-terminal", []string{"-e", "bash", "-c"}},
		{"tilix", []string{"-e", "bash", "-c"}},
	}

	script := `cd "$HOME" && clear && ` + absPath + `; echo; echo 'Press Enter to exit...'; read`

	for _, term := range terminals {
		if path, err := exec.LookPath(term.name); err == nil {
			args := append(term.args, script)
			cmd := exec.Command(path, args...)
			cmd.Start()
			return
		}
	}
}

// launchInTerminalWindows opens the app in Windows Terminal, PowerShell, or CMD
func launchInTerminalWindows(absPath string) {
	// Try Windows Terminal first (best experience)
	if wtPath, err := exec.LookPath("wt.exe"); err == nil {
		// Windows Terminal with PowerShell
		cmd := exec.Command(wtPath, "powershell", "-NoExit", "-Command", fmt.Sprintf("cd $HOME; Clear-Host; & '%s'; Read-Host 'Press Enter to exit'", absPath))
		cmd.Start()
		return
	}

	// Try PowerShell
	if psPath, err := exec.LookPath("powershell.exe"); err == nil {
		script := fmt.Sprintf("cd $HOME; Clear-Host; & '%s'; Read-Host 'Press Enter to exit'", absPath)
		cmd := exec.Command(psPath, "-NoExit", "-Command", script)
		cmd.Start()
		return
	}

	// Fallback to CMD
	if cmdPath, err := exec.LookPath("cmd.exe"); err == nil {
		script := fmt.Sprintf(`cd /d "%s" && cls && "%s" && pause`, os.Getenv("USERPROFILE"), absPath)
		cmd := exec.Command(cmdPath, "/k", script)
		cmd.Start()
	}
}
