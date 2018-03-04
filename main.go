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

// activateWindow switchet to the window and sends the key sequence to it.
func activateWindow(win string, keys []string) error {
	args := []string{"windowactivate", "--sync", win, "key", "--clearmodifiers"}
	args = append(args, keys...)
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
func editBuffer(buf []byte) ([]byte, error) {
	dir, err := ioutil.TempDir("", "diva-edit-")
	if err != nil {
		return nil, err
	}

	tempfile := filepath.Join(dir, "text")
	err = ioutil.WriteFile(tempfile, buf, 0600)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("gvim", "-f", tempfile)
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
func editClipboard() error {
	buf, err := getClipboard()
	if err != nil {
		return err
	}

	buf, err = editBuffer(buf)
	if err != nil {
		return err
	}

	return setClipboard(buf)
}

func main() {
	win, err := currentWindow()
	if err != nil {
		panic(err)
	}

	pid, err := windowPID(win)
	if err != nil {
		panic(err)
	}

	title, err := windowTitle(win)
	if err != nil {
		panic(err)
	}

	proc, err := ps.FindProcess(pid)
	if err != nil {
		panic(err)
	}

	cmd := proc.Executable()

	fmt.Printf("win: %v, cmd %v: %q\n", win, cmd, title)

	err = editClipboard()
	if err != nil {
		panic(err)
	}

	err = activateWindow(win, []string{"ctrl+v"})
	if err != nil {
		panic(err)
	}
}
