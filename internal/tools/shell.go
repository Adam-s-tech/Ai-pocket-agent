package tools

import (
	"context"
	"encoding/json"
	"time"
)

// commandTimeout bounds one command run for an agent.
const commandTimeout = 2 * time.Minute

func sshTool() *Tool {
	return &Tool{
		Name:     "ssh_run",
		Title:    "Run over SSH",
		Icon:     "terminal",
		Group:    "Terminal",
		Approval: true,
		Description: "Run a shell command on one of the computers the user saved in the app's Terminal, over SSH, and get its output. " +
			"Each call is a fresh non-interactive session: chain steps with && or ;, and don't start programs that wait for input.",
		Schema: object(map[string]any{
			"host":    str("The saved host's name."),
			"command": str("The command line to run."),
		}, "host", "command"),
		Run: func(ctx context.Context, env *Env, raw json.RawMessage) Result {
			var a struct{ Host, Command string }
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Shell == nil {
				return Errorf("SSH isn't available")
			}
			ctx, cancel := context.WithTimeout(ctx, commandTimeout)
			defer cancel()
			return commandResult(env.Shell.SSH(ctx, a.Host, a.Command))
		},
	}
}

func shellTool() *Tool {
	return &Tool{
		Name:     "shell_run",
		Title:    "Run in the device shell",
		Icon:     "square-terminal",
		Group:    "Terminal",
		Approval: true,
		Description: "Run a command in this device's own shell (Android's toybox tools), in the task's workspace folder, and get its output. " +
			"Not available on iPhone and iPad.",
		Schema: object(map[string]any{"command": str("The command line to run.")}, "command"),
		Run: func(ctx context.Context, env *Env, raw json.RawMessage) Result {
			var a struct{ Command string }
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Shell == nil {
				return Errorf("the shell isn't available")
			}
			ctx, cancel := context.WithTimeout(ctx, commandTimeout)
			defer cancel()
			return commandResult(env.Shell.Local(ctx, a.Command))
		},
	}
}

// commandResult reports a command's output, with its failure if any.
func commandResult(out string, err error) Result {
	if err != nil {
		if out == "" {
			return Errorf("%v", err)
		}
		return Result{Text: out + "\n[" + err.Error() + "]", IsError: true}
	}
	if out == "" {
		out = "(no output)"
	}
	return Result{Text: out}
}
