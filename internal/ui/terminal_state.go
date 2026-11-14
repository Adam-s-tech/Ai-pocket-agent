package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/term"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/xmhrnx/giode"
)

// termTab is an open terminal session.
type termTab struct {
	id    int
	t     *term.Term
	typed string // what the input field holds, already sent
	ctrl  bool   // the Ctrl key is held for the next key
	// jump scrolls the screen to the bottom on the next frame.
	jump bool
}

// knownHosts is the trusted SSH host keys.
func (a *App) knownHosts() *term.KnownHosts {
	if a.known == nil {
		a.known = &term.KnownHosts{Path: a.st.KnownHostsPath()}
	}
	return a.known
}

// sshKey returns the app's SSH private key, nil until one was made.
func (a *App) sshKey() []byte { return a.st.Secret(store.SSHKeyName) }

// sshPublicKey is the authorized_keys line for the app's key, "" if none.
func (a *App) sshPublicKey() string {
	k := a.sshKey()
	if len(k) == 0 {
		return ""
	}
	pub, err := term.PublicKey(k, "pocket-agent")
	if err != nil {
		return ""
	}
	return pub
}

// termHost is how to reach a saved host.
func (a *App) termHost(h store.SSHHost) term.Host {
	th := term.Host{Addr: h.Addr, Port: h.Port, User: h.User, Password: string(a.st.Secret("ssh/" + h.ID))}
	if h.UseKey {
		th.Key = a.sshKey()
	}
	return th
}

// trustPrompt asks the user, on the UI goroutine, whether to trust a
// host's key; it blocks the calling goroutine until they answer.
func (a *App) trustPrompt(host, fingerprint string) bool {
	answer := make(chan bool, 1)
	a.Post(func() {
		a.askOr(tr("Trust this computer?"),
			tr("First connection to {host}. Its key fingerprint is {fingerprint}. Trust it only if it matches the computer's own; you won't be asked again.", "host", host, "fingerprint", fingerprint),
			tr("Trust"), false, func() { answer <- true }, func() { answer <- false })
	})
	return <-answer
}

// openTerm adds a session and shows it.
func (a *App) openTerm(title string, sess term.Session) {
	a.nextTerm++
	tab := &termTab{id: a.nextTerm, jump: true}
	tab.t = term.New(title, sess, 80, 24, func() { a.Post(func() {}) })
	a.terms = append(a.terms, tab)
	a.navigate(fmt.Sprintf("terminal/session/%d", tab.id))
}

// connectHost opens a shell on a saved host.
func (a *App) connectHost(h store.SSHHost) {
	a.toast(giode.ToastInfo, tr("Connecting to {name}…", "name", h.Name))
	th := a.termHost(h)
	go func() {
		sess, err := term.DialSSH(context.Background(), th, a.knownHosts(), a.trustPrompt, 80, 24)
		a.Post(func() {
			if err != nil {
				a.toast(giode.ToastError, tr("Couldn't connect: {error}", "error", err))
				return
			}
			a.openTerm(h.Name, sess)
		})
	}()
}

// openLocalShell starts the device's shell in the app's files folder.
func (a *App) openLocalShell() {
	sess, err := term.Local(a.shellHome(), 80, 24)
	if err != nil {
		a.toast(giode.ToastError, err.Error())
		return
	}
	a.openTerm(tr("Local shell"), sess)
}

func (a *App) termTab(id int) *termTab {
	for _, t := range a.terms {
		if t.id == id {
			return t
		}
	}
	return nil
}

func (a *App) closeTerm(tab *termTab) {
	tab.t.Close()
	for i, t := range a.terms {
		if t == tab {
			a.terms = append(a.terms[:i], a.terms[i+1:]...)
			break
		}
	}
}

// appShell runs agents' commands: on saved SSH hosts, and in the
// device's shell in the conversation's workspace.
type appShell struct {
	a         *App
	hosts     []store.SSHHost
	workspace string
	resolve   func(store.SSHHost) term.Host
	known     *term.KnownHosts
}

func (s appShell) SSH(ctx context.Context, name, command string) (string, error) {
	for _, h := range s.hosts {
		if strings.EqualFold(h.Name, name) || h.ID == name {
			return term.RunSSH(ctx, s.resolve(h), s.known, command)
		}
	}
	var names []string
	for _, h := range s.hosts {
		names = append(names, h.Name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no SSH hosts are saved; the user adds them in the app's Terminal")
	}
	return "", fmt.Errorf("no saved host named %q; saved hosts: %s", name, strings.Join(names, ", "))
}

func (s appShell) Local(ctx context.Context, command string) (string, error) {
	return term.RunLocal(ctx, s.workspace, command)
}

// shellFor is the agents' shell for a turn, made on the UI goroutine.
func (a *App) shellFor(workspace string) appShell {
	hosts := append([]store.SSHHost(nil), a.settings.SSHHosts...)
	// Credentials are read now: the secure store is safe off the UI
	// goroutine, but the host list may change while the run goes on.
	resolved := map[string]term.Host{}
	for _, h := range hosts {
		resolved[h.ID] = a.termHost(h)
	}
	return appShell{a: a, hosts: hosts, workspace: workspace, known: a.knownHosts(),
		resolve: func(h store.SSHHost) term.Host { return resolved[h.ID] }}
}

// sshHostIndex lists the saved hosts for the system prompt, when the
// turn can use ssh_run.
func (a *App) sshHostIndex(set *tools.Set) [][2]string {
	if !hasTool(set, "ssh_run") {
		return nil
	}
	var out [][2]string
	for _, h := range a.settings.SSHHosts {
		out = append(out, [2]string{h.Name, h.User + "@" + h.Addr + portSuffix(h.Port)})
	}
	return out
}

// shellHome is the local shell's home folder, apart from the app's data.
func (a *App) shellHome() string {
	d := filepath.Join(a.st.Dir(), "home")
	_ = os.MkdirAll(d, 0o755)
	return d
}
