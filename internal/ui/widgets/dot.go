package widgets

import (
	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"
)

// Dot is a small status dot.
func Dot(color string, size int) El {
	return giode.Box(St{Width: pr.Width(size), Height: pr.Height(size), BorderRadius: pr.BorderRadius(size), Background: color})
}
