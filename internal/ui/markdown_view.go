package ui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/unit"

	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// markdown renders text as Markdown; key identifies its chunks. Put it
// in a SelectionArea to make it selectable.
// Long text is rendered in separately memoized chunks (see chunkSize).
func (a *App) markdown(key, text string) widgets.El { return a.markdownDoc(key, text, "") }

// markdownDoc is markdown for a document whose relative image paths are
// files in dir.
func (a *App) markdownDoc(key, text, dir string) widgets.El {
	scale := a.settings.FontScale
	if scale == 0 {
		scale = 1
	}
	size := float32(theme.SizeBody * scale)
	chunks := splitMarkdown(text, chunkSize)
	if len(chunks) == 1 {
		return a.markdownIn(key, text, size, dir)
	}
	kids := make([]widgets.El, len(chunks))
	for i, chunk := range chunks {
		ckey := fmt.Sprintf("%s/%d", key, i)
		kids[i] = giode.Memo("md/"+ckey, func() widgets.El { return a.markdownIn(ckey, chunk, size, dir) }, chunk, size, dir)
	}
	return widgets.Col(widgets.St{Gap: 12, Width: -1}, kids...)
}

func (a *App) markdownChunk(key, text string, size float32) widgets.El {
	return a.markdownIn(key, text, size, "")
}

// markdownIn renders Markdown whose relative image paths are files in
// dir, a conversation's workspace ("" for none).
func (a *App) markdownIn(key, text string, size float32, dir string) widgets.El {
	return giode.Markdown(text, giode.MarkdownOptions{
		LoadImage:      func(src string) (image.Image, error) { return loadMarkdownImage(src, dir) },
		Styles:         widgets.St{FontSize: unit.Sp(size), LineHeight: pr.LineHeight(size * 1.5), Color: theme.T.Text},
		CodeFont:       theme.FontMono,
		CodeBackground: theme.T.CodeBg,
		LinkColor:      theme.T.PrimaryText,
		OnLink:         a.openLink,
		Gap:            12,
	})
}

// openLink opens url in the browser, or copies it when it can't.
func (a *App) openLink(u string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.win.OpenURL(ctx, u); err != nil {
			a.Post(func() { a.copyText(u) })
		}
	}()
}

// loadMarkdownImage loads an image a reply shows: from the web, a data:
// URI, or a file in the workspace - never elsewhere on the device.
func loadMarkdownImage(src, dir string) (image.Image, error) {
	switch {
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"), strings.HasPrefix(src, "data:"):
	default:
		p, err := mediaPath(src, dir)
		if err != nil {
			return nil, err
		}
		src = p
	}
	img, err := giode.MarkdownFetchImage(src)
	if err != nil {
		return nil, err
	}
	return shrink(img, maxImageSide), nil
}
