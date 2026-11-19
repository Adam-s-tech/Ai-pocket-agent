package widgets

import (
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"
)

// IconTile is an icon on a tinted rounded square.
func IconTile(icon, color string, size int) El {
	if color == "" {
		color = theme.T.Primary
	}
	return giode.Box(St{
		Width: pr.Width(size), Height: pr.Height(size), BorderRadius: pr.BorderRadius(size / 4),
		Background: theme.Tint(color, 0.16), Align: giode.AlignCenter, Justify: giode.JustifyCenter,
	}, Icon(icon, size*3/5, color))
}
