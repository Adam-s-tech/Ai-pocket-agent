package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/term"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// terminalPage lists open sessions, the local shell, saved SSH hosts and
// the app's SSH key.
func (a *App) terminalPage(compact bool) widgets.El {
	add := widgets.IconButton("term/add", "plus", tr("Add a host"), 20, theme.T.Text, func() { a.navigate("terminal/host/new") })
	return a.pageScaffold("terminal", tr("Terminal"), compact, add, func() widgets.El {
		var kids []widgets.El
		if len(a.terms) > 0 {
			var rows []widgets.El
			for _, tab := range a.terms {
				tab := tab
				sub := tr("Connected")
				if ended, err := tab.t.Ended(); ended {
					sub = tr("Ended")
					if err != nil {
						sub = err.Error()
					}
				}
				rows = append(rows, widgets.ListRow(fmt.Sprintf("term/open/%d", tab.id), "square-terminal", "#34d399", tab.t.Title, sub,
					widgets.IconButton(fmt.Sprintf("term/close/%d", tab.id), "x", tr("Close"), 16, theme.T.Muted, func() { a.closeTerm(tab) }),
					func() { a.navigate(fmt.Sprintf("terminal/session/%d", tab.id)) }))
			}
			kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Open sessions")), widgets.Group(rows...)))
		}
		var rows []widgets.El
		if term.LocalShell {
			rows = append(rows, widgets.ListRow("term/local", "smartphone", "#a78bfa", tr("Local shell"), tr("This device's own shell"), nil, a.openLocalShell))
		}
		for _, h := range a.settings.SSHHosts {
			h := h
			rows = append(rows, widgets.ListRow("term/host/"+h.ID, "server", "#38bdf8", h.Name, h.User+"@"+h.Addr+portSuffix(h.Port),
				widgets.IconButton("term/edit/"+h.ID, "pencil", tr("Edit"), 16, theme.T.Muted, func() { a.navigate("terminal/host/" + h.ID) }),
				func() { a.connectHost(h) }))
		}
		if len(rows) == 0 {
			rows = append(rows, widgets.ListRow("term/first", "plus", theme.T.Primary, tr("Add a host"), tr("A computer or server to reach over SSH"), nil, func() { a.navigate("terminal/host/new") }))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Connect")), widgets.Group(rows...)))
		kids = append(kids, a.sshKeyCard())
		kids = append(kids, widgets.Text(tr("Agents can run commands on these hosts with the SSH tool, after asking you."), theme.SizeFootnote, theme.T.Muted))
		return widgets.Col(widgets.St{Gap: 18, Width: -1}, kids...)
	})
}

func portSuffix(p int) string {
	if p == 0 || p == 22 {
		return ""
	}
	return ":" + strconv.Itoa(p)
}

// sshKeyCard shows the app's public key, or offers to make one.
func (a *App) sshKeyCard() widgets.El {
	pub := a.sshPublicKey()
	if pub == "" {
		return widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("SSH key")), widgets.Card(widgets.St{Gap: 10},
			widgets.Text(tr("Sign in without a password: make a key, then add its public half to the server's ~/.ssh/authorized_keys."), theme.SizeFootnote, theme.T.Muted),
			widgets.Button("term/keygen", tr("Make a key"), "key-round", widgets.BtnPrimary, func() {
				priv, _, err := term.NewKey("pocket-agent")
				if err == nil {
					err = a.st.SetSecret(store.SSHKeyName, priv)
				}
				if err != nil {
					a.toast(giode.ToastError, err.Error())
				}
			})))
	}
	return widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("SSH key")), widgets.Card(widgets.St{Gap: 10},
		widgets.Text(tr("Add this line to ~/.ssh/authorized_keys on each server that should let this device in. The private key stays in this device's secure storage."), theme.SizeFootnote, theme.T.Muted),
		giode.Box(widgets.St{Width: -1, Padding: giode.UniformInset(10), BorderRadius: 10, Background: theme.T.CodeBg},
			widgets.CodeText("term/pub", pub, 12, theme.T.Text)),
		widgets.Row(widgets.St{Gap: 10, Width: -1},
			widgets.Button("term/copykey", tr("Copy"), "copy", widgets.BtnSecondary, func() { a.copyText(pub) }),
			widgets.Button("term/sharekey", tr("Share"), "share", widgets.BtnSecondary, func() { a.shareText(pub, "") }))))
}

// hostEditor adds or edits a saved SSH host.
func (a *App) hostEditor(id string, compact bool) widgets.El {
	creating := id == "new"
	h := store.SSHHost{UseKey: a.sshKey() != nil}
	if !creating {
		if found := a.settings.SSHHost(id); found != nil {
			h = *found
		}
	}
	k := "host/" + id + "/"
	name := a.input(k+"name", tr("Name, e.g. Home server"), h.Name)
	addr := a.input(k+"addr", tr("Address, e.g. 192.168.1.20 or example.com"), h.Addr, func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
	port := a.input(k+"port", "22", portText(h.Port), func(i *input.Input) { i.Keyboard(giode.KeyboardNumeric) })
	user := a.input(k+"user", tr("User name"), h.User)
	pass := a.input(k+"pass", tr("Password (optional with a key)"), "", func(i *input.Input) { i.Password(true) })
	useKey := a.hostUseKey(k, h.UseKey)
	save := func() {
		h.Name = strings.TrimSpace(name.Text())
		h.Addr = strings.TrimSpace(addr.Text())
		h.User = strings.TrimSpace(user.Text())
		h.Port, _ = strconv.Atoi(strings.TrimSpace(port.Text()))
		h.UseKey = *useKey
		if h.Addr == "" || h.User == "" {
			a.toast(giode.ToastWarning, tr("Enter the address and user name"))
			return
		}
		if h.Name == "" {
			h.Name = h.User + "@" + h.Addr
		}
		if creating {
			h.ID = store.NewID()
			a.settings.SSHHosts = append(a.settings.SSHHosts, h)
		} else if found := a.settings.SSHHost(id); found != nil {
			*found = h
		}
		if p := pass.Text(); p != "" {
			if err := a.st.SetSecret("ssh/"+h.ID, []byte(p)); err != nil {
				a.toast(giode.ToastError, err.Error())
			}
		}
		a.saveSettings()
		for _, f := range []string{"name", "addr", "port", "user", "pass", "key"} {
			a.dropInput(k + f)
		}
		delete(a.hostKeys, k)
		a.back()
	}
	kids := []widgets.El{
		widgets.Card(widgets.St{Gap: 12}, name, addr, port, user, pass,
			widgets.ListRow(k+"usekey", "key-round", "#f59e0b", tr("Sign in with this device's SSH key"), tr("Set it up in the Terminal page"),
				widgets.Toggle(k+"usekey/t", *useKey, func(v bool) { *useKey = v }), nil)),
		widgets.Button(k+"save", ifStr(creating, tr("Add host"), tr("Save")), "check", widgets.BtnPrimary, save),
	}
	if !creating {
		kids = append(kids,
			widgets.Button(k+"forget", tr("Forget its key fingerprint"), "rotate-ccw", widgets.BtnSecondary, func() {
				a.knownHosts().Forget(term.Host{Addr: h.Addr, Port: h.Port}.Address())
				a.toast(giode.ToastSuccess, tr("You'll be asked to check its key on the next connection"))
			}),
			widgets.Button(k+"delete", tr("Remove host"), "trash-2", widgets.BtnDanger, func() {
				a.ask(tr("Remove {name}?", "name", h.Name), tr("Its saved password is deleted from this device."), tr("Remove"), true, func() {
					for i, x := range a.settings.SSHHosts {
						if x.ID == h.ID {
							a.settings.SSHHosts = append(a.settings.SSHHosts[:i], a.settings.SSHHosts[i+1:]...)
							break
						}
					}
					_ = a.st.SetSecret("ssh/"+h.ID, nil)
					a.saveSettings()
					a.back()
				})
			}))
	}
	return a.pageScaffold("terminal/host/"+id, ifStr(creating, tr("Add a host"), h.Name), compact, nil, func() widgets.El {
		return widgets.Col(widgets.St{Gap: 14, Width: -1}, kids...)
	})
}

// hostUseKey is the editor's key toggle, kept across frames.
func (a *App) hostUseKey(k string, init bool) *bool {
	if a.hostKeys == nil {
		a.hostKeys = map[string]*bool{}
	}
	if v, ok := a.hostKeys[k]; ok {
		return v
	}
	v := init
	a.hostKeys[k] = &v
	return &v
}

func portText(p int) string {
	if p == 0 {
		return ""
	}
	return strconv.Itoa(p)
}
