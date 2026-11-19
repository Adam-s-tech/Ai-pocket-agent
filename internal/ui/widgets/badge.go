package widgets

import (
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/xmhrnx/giode"
)

// Badge is a small static label.
func Badge(label, fg, bg string) El {
	return giode.Box(St{Padding: giode.SymmetricInset(7, 2), BorderRadius: 6, Background: bg},
		Text(label, theme.SizeMicro, fg, giode.FontWeightMedium))
}
