package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ps "github.com/mitchellh/go-ps"
)

// currentWindow finds the ID of the currently active window.
func currentWindow() (string, error) {
	cmd := exec.Command("xdotool", "getactivewindow")
	cmd.Stderr = os.Stderr

	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}

// windowPID returns the process ID belonging to the window.
func windowPID(win string) (int, error) {
	cmd := exec.Command("xdotool", "getwindowpid", win)
	cmd.Stderr = os.Stderr

	buf, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	id, err := strconv.Atoi(strings.TrimSpace(string(buf)))
	if err != nil {
		return 0, err
	}

	return id, nil
}

// windowTitle returns the title for the window id.
func windowTitle(win string) (string, error) {
	cmd := exec.Command("xdotool", "getwindowname", win)
	cmd.Stderr = os.Stderr

	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}

// sendKeys sends the key sequence to the window, with delay in between the keys.
func sendKeys(win string, delay time.Duration, keys []string) error {
	// disable keyboard repeat
	cmd := exec.Command("xset", "r", "off")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}

	for _, key := range keys {
		args := []string{"key", "--clearmodifiers", "--window", win, key}
		fmt.Printf("running xdotool %v\n", args)
		cmd = exec.Command("xdotool", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return err
		}

		time.Sleep(delay)
	}

	// reenable keyboard repeat
	cmd = exec.Command("xset", "r", "on")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// activateWindow switches to the window and sends the key sequence to it.
func activateWindow(win string) error {
	// switch to the previous window and paste the buffer
	args := []string{"windowactivate", "--sync", win}
	fmt.Printf("running xdotool %v\n", args)
	cmd := exec.Command("xdotool", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// getClipboard returns the contents of the clipboard.
func getClipboard() ([]byte, error) {
	cmd := exec.Command("xclip", "-out", "-selection", "clipboard")
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// setClipboard sets the contents of the clipboard to buf.
func setClipboard(buf []byte) error {
	cmd := exec.Command("xclip", "-in", "-selection", "clipboard")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = bytes.NewReader(buf)
	return cmd.Run()
}

// editBuffer runs gvim on a temp file and returns the new buffer
func editBuffer(fileExt string, buf []byte) ([]byte, error) {
	dir, err := ioutil.TempDir("", "diva-edit-")
	if err != nil {
		return nil, err
	}

	tempfile := filepath.Join(dir, "text"+fileExt)
	err = ioutil.WriteFile(tempfile, buf, 0600)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("gvim", "-f", "-c", ":Goyo", "-c", ":PencilSoft", tempfile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	buf, err = ioutil.ReadFile(tempfile)
	if err != nil {
		return nil, err
	}

	err = os.Remove(tempfile)
	if err != nil {
		return nil, err
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// editClipboard runs an editor on the clipboard contents.
func editClipboard(fileExt string) error {
	buf, err := getClipboard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get clipboard, using empty buffer: %v\n", err)
		buf = []byte("")
	}

	buf, err = editBuffer(fileExt, buf)
	if err != nil {
		return err
	}

	return setClipboard(buf)
}

// really simple pattern matching to select the extension for the text file
// based on window title and executable.
var pattern = []struct {
	cmd   string
	title string
	ext   string
}{
	{
		cmd: "chromium",
		ext: ".md",
	},
	{
		cmd: "firefox",
		ext: ".md",
	},
}

func findExtension(cmd, title string) string {
	for _, pat := range pattern {
		if pat.cmd != "" && !strings.Contains(cmd, pat.cmd) {
			continue
		}

		if pat.title != "" && !strings.Contains(title, pat.title) {
			continue
		}

		return pat.ext
	}

	// default
	return ".txt"
}

// die prints the message to stderr and exits.
func die(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func main() {
	win, err := currentWindow()
	if err != nil {
		die("unable to find current window")
	}

	pid, err := windowPID(win)
	if err != nil {
		die("unable to find PID for window %v", win)
	}

	title, err := windowTitle(win)
	if err != nil {
		die("unable to find title for window %v", win)
	}

	proc, err := ps.FindProcess(pid)
	if err != nil {
		die("unable to find process for PID %v", pid)
	}

	cmd := proc.Executable()

	fmt.Printf("win: %v, cmd %v: %q\n", win, cmd, title)

	err = sendKeys(win, 0, []string{"ctrl+a", "ctrl+c"})
	if err != nil {
		die("copying text failed: %v", err)
	}

	err = editClipboard(findExtension(proc.Executable(), title))
	if err != nil {
		fmt.Fprintf(os.Stderr, "editing clipboard failed: %v\nswitching back to window %v", err, win)
		err = activateWindow(win)
		if err != nil {
			die("switching back to window %v failed: %v", win, err)
		}
		return
	}

	err = activateWindow(win)
	if err != nil {
		die("switching back to window %v failed: %v", win, err)
	}
}
