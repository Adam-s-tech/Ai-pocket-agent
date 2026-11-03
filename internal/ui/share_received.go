package ui

import (
	"strings"

	"github.com/xmhrnx/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// ShareReceived starts a new chat with what another app shared - text,
// a link, files - in the composer, ready to ask about. UI goroutine.
func (a *App) ShareReceived(s giode.SharedContent) {
	a.newChat(store.ModeChat, "")
	var parts []string
	for _, p := range []string{s.Subject, s.Text} {
		if p = strings.TrimSpace(p); p != "" && !strings.Contains(strings.Join(parts, "\n"), p) {
			parts = append(parts, p)
		}
	}
	a.composer.load(strings.Join(parts, "\n\n"), nil)
	for _, f := range s.Files {
		a.attachPicked(f)
	}
	a.composer.focus()
}
