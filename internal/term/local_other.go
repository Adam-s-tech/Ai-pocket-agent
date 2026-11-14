//go:build !((linux || darwin || freebsd) && !ios)

package term

import "context"

// LocalShell reports whether this platform can run a local shell.
const LocalShell = false

// Local is unavailable here: iOS doesn't let apps start programs.
func Local(dir string, cols, rows int) (Session, error) { return nil, ErrNoLocalShell }

// RunLocal is unavailable here.
func RunLocal(ctx context.Context, dir, command string) (string, error) { return "", ErrNoLocalShell }
