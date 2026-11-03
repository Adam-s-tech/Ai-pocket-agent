package widgets

import (
	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"
)

// Centered lays child out at the full available width up to maxWidth
// dp, centered horizontally: a readable content column.
func Centered(maxWidth int, child El) El {
	return giode.Box(St{Width: -1, Justify: giode.JustifyCenter},
		giode.Box(St{Width: -1, MaxWidth: pr.MaxWidth(maxWidth), FlexDirection: giode.FlexDirectionColumn}, child))
}
