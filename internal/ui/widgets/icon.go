package widgets

import (
	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"
)

// Icon renders a registered icon.
func Icon(name string, size int, color string) El {
	return giode.Icon(name, St{Width: pr.Width(size), Height: pr.Height(size), Color: color})
}
