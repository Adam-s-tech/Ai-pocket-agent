package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// confirmState is a yes/no question.
type confirmState struct {
	title, message, action string
	danger                 bool
	onYes                  func()
	// onNo runs once if the question closes without a yes.
	onNo func()
}

// ask shows a confirmation; onYes runs if the user agrees.
func (a *App) ask(title, message, action string, danger bool, onYes func()) {
	a.askOr(title, message, action, danger, onYes, nil)
}

// askOr is ask with onNo, run when the question is cancelled or
// dismissed instead.
func (a *App) askOr(title, message, action string, danger bool, onYes, onNo func()) {
	if c := a.confirm; c != nil && c.onNo != nil {
		c.onNo() // a question replaced unanswered is a no
	}
	a.confirm = &confirmState{title: title, message: message, action: action, danger: danger, onYes: onYes, onNo: onNo}
	a.confirmOpen = true
}

func (a *App) confirmView() widgets.El {
	if a.confirmPanel == nil {
		a.confirmPanel = newSheet(&a.confirmOpen, func() widgets.El {
			c := a.confirm
			if c == nil {
				return giode.Box(widgets.St{})
			}
			kind := widgets.BtnPrimary
			if c.danger {
				kind = widgets.BtnDanger
			}
			return widgets.Col(widgets.St{Gap: 18, Width: -1, Padding: giode.Inset{Bottom: 8}},
				widgets.Text(c.message, theme.SizeCallout, theme.T.Text2),
				widgets.Row(widgets.St{Gap: 10, Width: -1},
					widgets.Spacer(),
					widgets.Button("confirm/no", tr("Cancel"), "", widgets.BtnSecondary, func() { a.confirmOpen = false }),
					widgets.Button("confirm/yes", c.action, "", kind, func() {
						a.confirmOpen = false
						c.onNo = nil
						c.onYes()
					}),
				),
			)
		})
	}
	if c := a.confirm; c != nil && !a.confirmOpen && c.onNo != nil {
		no := c.onNo
		c.onNo = nil
		no()
	}
	if c := a.confirm; c != nil {
		a.confirmPanel.Title(c.title).Styles(widgets.St{Background: theme.T.Elevated, Color: theme.T.Text})
	}
	return a.confirmPanel
}
