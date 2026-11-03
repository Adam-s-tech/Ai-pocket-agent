package widgets

import (
	"github.com/xmhrnx/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Badge is a small static label.
func Badge(label, fg, bg string) El {
	return giode.Box(St{Padding: giode.SymmetricInset(7, 2), BorderRadius: 6, Background: bg},
		Text(label, theme.SizeMicro, fg, giode.FontWeightMedium))
}
