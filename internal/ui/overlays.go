package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// overlays holds the app's sheet and confirmation dialog; they take no
// space and paint over the window while open.
func (a *App) overlays() widgets.El {
	return giode.Box(widgets.St{}, a.sheetView(), a.confirmView())
}
