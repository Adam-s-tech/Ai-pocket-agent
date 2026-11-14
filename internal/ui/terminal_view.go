package ui

import (
	"fmt"
	"strings"

	gkey "gioui.org/io/key"
	"gioui.org/unit"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/input"
	pr "github.com/xmhrnx/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/term"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// Terminal text: JetBrains Mono's advance is 0.6 em.
const (
	termFont    = 13
	termLineH   = 17
	termAdvance = 0.6
	termBg      = "#0b0c0f"
	termFg      = "#d8dce4"
)

// termView shows a session full screen: its output, a row of keys a
// phone keyboard lacks, and a field to type in.
func (a *App) termView(id int, compact bool) widgets.El {
	tab := a.termTab(id)
	if tab == nil {
		return a.pageScaffold("terminal/gone", tr("Terminal"), compact, nil, func() widgets.El {
			return widgets.Text(tr("This session was closed."), theme.SizeBody, theme.T.Muted)
		})
	}
	t := tab.t
	ended, err := t.Ended()
	status := ""
	switch {
	case ended && err != nil:
		status = err.Error()
	case ended:
		status = tr("Session ended")
	}
	header := widgets.Row(widgets.St{Height: 52, Padding: giode.SymmetricInset(8, 0), Gap: 4, Width: -1},
		widgets.IconButton("term/back", "chevron-left", tr("Back"), 22, theme.T.Text, a.back),
		giode.Box(widgets.St{FlexGrow: 1}, widgets.Col(widgets.St{Gap: 0},
			widgets.Line(t.Title, theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold),
			widgets.When(status != "", func() widgets.El { return widgets.Line(status, theme.SizeCaption, theme.T.Muted) }))),
		widgets.IconButton("term/paste", "clipboard-paste", tr("Paste"), 19, theme.T.Text2, func() {
			a.win.ReadClipboard(func(text string) { a.Post(func() { t.Paste(text) }) })
		}),
		widgets.IconButton("term/x", "x", tr("Close"), 20, theme.T.Text2, func() {
			a.closeTerm(tab)
			a.back()
		}),
	)
	screen := widgets.MeasureSize(a.termScreen(tab), func(w, h, dpPerSp float32) {
		cell := termFont * termAdvance * dpPerSp
		line := termLineH * dpPerSp
		t.Resize(int((w-16)/cell), int((h-8)/line))
	})
	return widgets.Col(widgets.St{Width: -1, Height: -1, Background: termBg},
		widgets.Unselectable(header),
		giode.Box(widgets.St{FlexGrow: 1, Width: -1, FlexDirection: giode.FlexDirectionColumn}, screen),
		a.termKeys(tab),
		a.termInput(tab),
	)
}

// termScreen is the scrollback and screen as a list kept at the bottom.
func (a *App) termScreen(tab *termTab) widgets.El {
	t := tab.t
	list := giode.Keyed("term/list", fmt.Sprint(tab.id), func() *giode.LazyListElement {
		return giode.LazyList(
			func() int { return t.Lines() },
			func(i int) any { return i },
			func(i int) giode.Element { return termRow(t.Line(i)) },
			widgets.St{ScrollBar: scrollBar(), Padding: giode.Inset{Left: 8, Right: 8, Top: 4, Bottom: 4}},
		).Version(func(i int) any { return t.Version() }).Anchor(giode.LazyAnchorBottom)
	})
	if tab.jump {
		tab.jump = false
		list.ScrollToBottom()
	}
	return giode.Box(widgets.St{Width: -1, FlexGrow: 1, FlexDirection: giode.FlexDirectionColumn}, list)
}

// termRow draws one line of runs in the terminal's font.
func termRow(runs []term.Run) giode.Element {
	st := widgets.St{FontFamily: theme.FontMono, FontSize: unit.Sp(termFont), LineHeight: pr.LineHeight(termLineH), Color: termFg,
		MaxLines: 1, Height: termLineH}
	if len(runs) == 0 {
		return giode.Box(widgets.St{Height: termLineH, Width: -1})
	}
	spans := make([]giode.RichTextSpan, 0, len(runs))
	for _, r := range runs {
		fg, bg := termColor(r.FG, termFg), termColor(r.BG, "")
		if r.Cursor {
			fg, bg = termBg, termColor(r.FG, termFg)
		}
		sst := giode.Styles{Color: pr.Color(fg)}
		if bg != "" {
			sst.Background = pr.Color(bg)
		}
		if r.Bold {
			sst.FontWeight = giode.FontWeightBold
		}
		spans = append(spans, giode.Span(r.Text, sst))
	}
	return giode.RichText(st, spans...)
}

// termColor maps a run's color: "" is the default, "fg" and "bg" the
// default foreground and background swapped in by reverse video.
func termColor(c, def string) string {
	switch c {
	case "":
		return def
	case "fg":
		return termFg
	case "bg":
		return termBg
	}
	return c
}

// termKeys are the keys phone keyboards lack.
func (a *App) termKeys(tab *termTab) widgets.El {
	t := tab.t
	press := func(code rune) func() {
		return func() {
			mod := uv.KeyMod(0)
			if tab.ctrl {
				mod, tab.ctrl = uv.ModCtrl, false
			}
			t.SendKey(code, mod)
		}
	}
	keys := []widgets.El{
		widgets.Chip("tk/esc", "Esc", "", false, press(uv.KeyEscape)),
		widgets.Chip("tk/tab", "Tab", "", false, press(uv.KeyTab)),
		widgets.Chip("tk/ctrl", "Ctrl", "", tab.ctrl, func() { tab.ctrl = !tab.ctrl }),
		widgets.Chip("tk/up", "↑", "", false, press(uv.KeyUp)),
		widgets.Chip("tk/down", "↓", "", false, press(uv.KeyDown)),
		widgets.Chip("tk/left", "←", "", false, press(uv.KeyLeft)),
		widgets.Chip("tk/right", "→", "", false, press(uv.KeyRight)),
		widgets.Chip("tk/home", "Home", "", false, press(uv.KeyHome)),
		widgets.Chip("tk/end", "End", "", false, press(uv.KeyEnd)),
		widgets.Chip("tk/pgup", "PgUp", "", false, press(uv.KeyPgUp)),
		widgets.Chip("tk/pgdn", "PgDn", "", false, press(uv.KeyPgDown)),
	}
	for _, s := range []string{"|", "/", "-", "~", "$", "&", ">", "*"} {
		s := s
		keys = append(keys, widgets.Chip("tk/"+s, s, "", false, func() { t.SendText(s) }))
	}
	sc := a.hscroll("term/keys", func() widgets.El { return widgets.Row(widgets.St{Gap: 6}, keys...) })
	return widgets.Unselectable(giode.Box(widgets.St{Width: -1, Padding: giode.SymmetricInset(8, 6)}, sc))
}

// termInput is where the user types: text goes to the session as it is
// typed, Enter sends a return, and deleting sends backspaces.
func (a *App) termInput(tab *termTab) widgets.El {
	t := tab.t
	k := fmt.Sprintf("term/in/%d", tab.id)
	in, ok := a.inputs[k]
	if !ok {
		in = a.input(k, tr("Type here; Enter sends"), "", func(i *input.Input) { i.Autocorrect(false) })
		in.OnChange(func(text string) { a.termTyped(tab, text) })
		in.Submit(func(string) {
			t.SendKey(uv.KeyEnter, 0)
			tab.typed = ""
			in.SetText("")
		})
		in.OnKeys(func(e gkey.Event) {
			if e.State != gkey.Press {
				return
			}
			switch e.Name {
			case gkey.NameDeleteBackward:
				if r := []rune(tab.typed); len(r) > 0 {
					tab.typed = string(r[:len(r)-1])
					in.SetText(tab.typed)
				}
				t.SendKey(uv.KeyBackspace, 0)
			case gkey.NameTab:
				t.SendKey(uv.KeyTab, 0)
			case gkey.NameEscape:
				t.SendKey(uv.KeyEscape, 0)
			case gkey.NameUpArrow:
				t.SendKey(uv.KeyUp, 0)
			case gkey.NameDownArrow:
				t.SendKey(uv.KeyDown, 0)
			}
		}, gkey.NameDeleteBackward, gkey.NameTab, gkey.NameEscape, gkey.NameUpArrow, gkey.NameDownArrow)
		in.Focus()
	}
	return giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 8, Right: 8, Bottom: 8}}, in)
}

// termTyped sends what changed in the field since last time: new text,
// or backspaces for text removed (a keyboard's autocorrect may do both).
func (a *App) termTyped(tab *termTab, text string) {
	prev := []rune(tab.typed)
	cur := []rune(text)
	common := 0
	for common < len(prev) && common < len(cur) && prev[common] == cur[common] {
		common++
	}
	for range prev[common:] {
		tab.t.SendKey(uv.KeyBackspace, 0)
	}
	added := string(cur[common:])
	if tab.ctrl && added != "" {
		// Ctrl applies to the first key typed after it.
		r := []rune(strings.ToLower(added))
		tab.t.SendKey(r[0], uv.ModCtrl)
		tab.ctrl = false
		added = string(r[1:])
	}
	if added != "" {
		tab.t.SendText(added)
	}
	tab.typed = text
}
