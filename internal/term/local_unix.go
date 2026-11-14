//go:build (linux || darwin || freebsd) && !ios

package term

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// LocalShell reports whether this platform can run a local shell.
const LocalShell = true

// shell is the platform's shell.
func shell() string {
	if runtime.GOOS == "android" {
		return "/system/bin/sh"
	}
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

type localSession struct {
	f   *os.File
	cmd *exec.Cmd
}

// keySignals are the signals a terminal's Ctrl+C, Ctrl+\ and Ctrl+Z send.
var keySignals = []os.Signal{syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTSTP}

var unignore sync.Once

// unignoreKeySignals stops the shell inheriting ignored key signals: an
// app started with them ignored passes that on to the programs it
// starts, and Ctrl+C then interrupts nothing. Go installs a handler for
// an ignored signal once it's notified of it, and started programs get
// the default action for a handled one.
func unignoreKeySignals() {
	unignore.Do(func() {
		for _, sig := range keySignals {
			if signal.Ignored(sig) {
				signal.Notify(make(chan os.Signal, 1), sig)
			}
		}
	})
}

// Local starts the device's shell in dir, in a cols x rows terminal.
func Local(dir string, cols, rows int) (Session, error) {
	unignoreKeySignals()
	cmd := exec.Command(shell())
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "HOME="+dir)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	return &localSession{f: f, cmd: cmd}, nil
}

func (s *localSession) Read(p []byte) (int, error)  { return s.f.Read(p) }
func (s *localSession) Write(p []byte) (int, error) { return s.f.Write(p) }
func (s *localSession) Resize(cols, rows int) error {
	return pty.Setsize(s.f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}
func (s *localSession) Close() error {
	_ = s.cmd.Process.Kill()
	return s.f.Close()
}

// RunLocal runs command with the device's shell in dir and returns its
// combined output, cut to maxOutput.
func RunLocal(ctx context.Context, dir, command string) (string, error) {
	cmd := exec.CommandContext(ctx, shell(), "-c", command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+dir)
	out, err := cmd.CombinedOutput()
	return clip(out), err
}
