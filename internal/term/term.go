// Package term runs terminal sessions - SSH to a remote host, or a local
// shell where the platform allows one - through a VT emulator the UI
// draws, and runs one-off commands for agents.
package term

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// Session is what a terminal shows: output to Read, input to Write.
type Session interface {
	io.ReadWriteCloser
	// Resize tells the other end the terminal's size in cells.
	Resize(cols, rows int) error
}

// ErrNoLocalShell is returned where the platform can't run programs:
// iOS doesn't let apps start processes.
var ErrNoLocalShell = errors.New("this device can't run a local shell; connect to a computer over SSH instead")

// scrollbackLines is how much output scrolled off the top is kept.
const scrollbackLines = 5000

// Term is a session shown through a terminal emulator.
type Term struct {
	Title string

	emu  *vt.SafeEmulator
	sess Session
	// version counts changes, for redrawing.
	version atomic.Uint64

	mu     sync.Mutex
	err    error
	closed bool
	cols   int
	rows   int
}

// New shows sess in a terminal of cols x rows cells. onChange runs, from
// another goroutine, when the screen changes or the session ends.
func New(title string, sess Session, cols, rows int, onChange func()) *Term {
	t := &Term{Title: title, emu: vt.NewSafeEmulator(cols, rows), sess: sess, cols: cols, rows: rows}
	t.emu.SetScrollbackSize(scrollbackLines)
	go func() { // the session's output, into the screen
		buf := make([]byte, 32<<10)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				_, _ = t.emu.Write(buf[:n])
				t.version.Add(1)
				onChange()
			}
			if err != nil {
				t.mu.Lock()
				if !errors.Is(err, io.EOF) {
					t.err = err
				}
				t.closed = true
				t.mu.Unlock()
				t.version.Add(1)
				onChange()
				return
			}
		}
	}()
	go func() { // keys and the emulator's replies, to the session
		buf := make([]byte, 4<<10)
		for {
			n, err := t.emu.Read(buf)
			if n > 0 {
				_, _ = sess.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return t
}

// Version changes whenever the screen does.
func (t *Term) Version() uint64 { return t.version.Load() }

// Ended reports whether the session ended, and why if it failed.
func (t *Term) Ended() (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed, t.err
}

// Close ends the session.
func (t *Term) Close() {
	_ = t.sess.Close()
	_ = t.emu.Close()
}

// SendText types text, as a keyboard would.
func (t *Term) SendText(s string) { t.emu.SendText(s) }

// SendKey presses a key, encoded for the program's current modes (the
// arrows differ in application cursor mode, as vim and less use).
func (t *Term) SendKey(code rune, mod uv.KeyMod) {
	t.emu.SendKey(uv.KeyPressEvent{Code: code, Mod: mod})
}

// Paste pastes text, bracketed when the program asked for that.
func (t *Term) Paste(s string) { t.emu.Paste(s) }

// Resize changes the terminal's size in cells.
func (t *Term) Resize(cols, rows int) {
	cols, rows = max(cols, 20), max(rows, 5)
	t.mu.Lock()
	same := cols == t.cols && rows == t.rows
	t.cols, t.rows = cols, rows
	t.mu.Unlock()
	if same {
		return
	}
	t.emu.Resize(cols, rows)
	_ = t.sess.Resize(cols, rows)
	t.version.Add(1)
}

// Size is the terminal's size in cells.
func (t *Term) Size() (cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cols, t.rows
}

// Lines is how many lines there are to show: the scrollback, then the
// screen. The alternate screen (full-screen programs) has no scrollback.
func (t *Term) Lines() int {
	if t.emu.IsAltScreen() {
		return t.emu.Height()
	}
	return t.emu.ScrollbackLen() + t.emu.Height()
}

// Run is a stretch of a line in one style. Colors are "#rrggbb", or ""
// for the terminal's default.
type Run struct {
	Text   string
	FG, BG string
	Bold   bool
	Cursor bool
}

// Line returns line i of Lines, as styled runs; trailing blanks are
// dropped.
func (t *Term) Line(i int) []Run {
	sb := 0
	if !t.emu.IsAltScreen() {
		sb = t.emu.ScrollbackLen()
	}
	w := t.emu.Width()
	cur := t.emu.CursorPosition()
	showCursor := !t.emu.CursorHidden()
	var cells []*uv.Cell
	y := i - sb
	for x := 0; x < w; x++ {
		if y < 0 {
			cells = append(cells, t.emu.ScrollbackCellAt(x, i))
		} else {
			cells = append(cells, t.emu.CellAt(x, y))
		}
	}
	cursorX := -1
	if y >= 0 && showCursor && cur.Y == y {
		cursorX = cur.X
	}
	return runs(cells, cursorX)
}

// runs groups cells into styled runs, marking the cursor's cell.
func runs(cells []*uv.Cell, cursorX int) []Run {
	end := len(cells)
	for end > 0 && end-1 != cursorX && blank(cells[end-1]) {
		end--
	}
	var out []Run
	for x := 0; x < end; x++ {
		c := cells[x]
		r := Run{Text: " "}
		if c != nil {
			if c.Width == 0 && c.Content == "" {
				continue // the second half of a wide character
			}
			if c.Content != "" {
				r.Text = c.Content
			}
			r.FG, r.BG = hex(c.Style.Fg), hex(c.Style.Bg)
			r.Bold = c.Style.Attrs&uv.AttrBold != 0
			if c.Style.Attrs&uv.AttrReverse != 0 {
				r.FG, r.BG = r.BG, r.FG
				if r.FG == "" {
					r.FG = "bg"
				}
				if r.BG == "" {
					r.BG = "fg"
				}
			}
		}
		r.Cursor = x == cursorX
		if n := len(out); n > 0 && !r.Cursor && !out[n-1].Cursor && out[n-1].FG == r.FG && out[n-1].BG == r.BG && out[n-1].Bold == r.Bold {
			out[n-1].Text += r.Text
			continue
		}
		out = append(out, r)
	}
	return out
}

func blank(c *uv.Cell) bool {
	return c == nil || (strings.TrimSpace(c.Content) == "" && c.Style.Bg == nil && c.Style.Attrs&uv.AttrReverse == 0)
}

// palette is the 16 standard colors, tuned to read on a dark
// background (the xterm defaults' blue is near black).
var palette = [16]string{
	"#1d1f24", "#f07178", "#a6da95", "#eed49f", "#7aa2f7", "#c6a0f6", "#7dcfff", "#c0c6d0",
	"#5b6270", "#ff8a8f", "#b8f0a8", "#ffe6a8", "#9ab8ff", "#dcbcff", "#a0e2ff", "#f2f4f8",
}

// hex turns a color into "#rrggbb", "" for none.
func hex(c color.Color) string {
	switch c := c.(type) {
	case nil:
		return ""
	case ansi.BasicColor:
		if c < 16 {
			return palette[c]
		}
	case ansi.IndexedColor:
		if c < 16 {
			return palette[c]
		}
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
