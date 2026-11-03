package widgets

import (
	"github.com/xmhrnx/giode"
	pr "github.com/xmhrnx/giode/styles/properties"
)

// Avatar shows a name's initials on a stable color, size dp across.
func Avatar(name string, size int) El {
	return giode.Avatar(name, St{Width: pr.Width(size)})
}
