package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
	"github.com/xmhrnx/giode"
)

// fileViewer shows a workspace file: Markdown rendered, other text as
// code.
func (a *App) fileViewer(spec string, compact bool) widgets.El {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 {
		return a.pageScaffold("file", tr("File"), compact, nil, func() widgets.El { return widgets.Text(tr("Not found"), theme.SizeBody, theme.T.Muted) })
	}
	convID, rel := parts[0], parts[1]
	full := filepath.Join(a.st.Workspace(convID), filepath.FromSlash(rel))
	actions := widgets.Row(widgets.St{Gap: 2},
		widgets.IconButton("fv/copy", "copy", tr("Copy"), 18, theme.T.Text2, func() {
			if b, err := os.ReadFile(full); err == nil {
				a.copyText(string(b))
			}
		}),
		widgets.IconButton("fv/open", "external-link", tr("Open in another app"), 18, theme.T.Text2, func() { a.openFile(full) }),
		widgets.IconButton("fv/share", "share", tr("Share"), 18, theme.T.Text2, func() { a.shareFile(full, mimeOf(rel)) }),
	)
	return a.pageScaffold("file/"+spec, lastPathElem(rel), compact, actions, func() widgets.El {
		switch kind := mediaKind(rel); kind {
		case "image":
			return a.chatImage("fv/img/"+spec, rel, "", a.st.Workspace(convID))
		case "video", "audio":
			return a.mediaCard("fv/media/"+spec, kind, rel, "", a.st.Workspace(convID))
		}
		b, err := os.ReadFile(full)
		if err != nil {
			return widgets.Text(tr("This file no longer exists."), theme.SizeBody, theme.T.Muted)
		}
		if len(b) > 400_000 {
			b = append(b[:400_000], []byte(tr("\n\n… (truncated)"))...)
		}
		info := fmt.Sprintf(tr("%s · %s"), rel, tools.HumanBytes(int64(len(b))))
		var body widgets.El
		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			dir := filepath.Dir(full)
			body = giode.SelectionArea("file/"+spec, a.markdownDoc("file/"+spec, string(b), dir))
		} else {
			body = giode.Box(widgets.St{Width: -1, Padding: giode.UniformInset(14), BorderRadius: 12, Background: theme.T.CodeBg, BorderWidth: 1, BorderColor: theme.T.CodeBorder},
				widgets.CodeText("file/"+spec, string(b), 13, theme.T.Text))
		}
		return widgets.Col(widgets.St{Gap: 14, Width: -1}, widgets.Text(info, theme.SizeCaption, theme.T.Muted), body)
	})
}
