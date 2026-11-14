package widgets

import "gioui.org/layout"

// MeasureSize lays out el and reports the size it was given, in dp, and
// how many dp an sp is (the system's font scale), for fitting a grid of
// text into it.
func MeasureSize(el El, size func(w, h, dpPerSp float32)) El { return measureSize{el, size} }

type measureSize struct {
	el   El
	size func(w, h, dpPerSp float32)
}

func (m measureSize) Layout(gtx layout.Context) layout.Dimensions {
	px := gtx.Metric.PxPerDp
	m.size(float32(gtx.Constraints.Max.X)/px, float32(gtx.Constraints.Max.Y)/px, gtx.Metric.PxPerSp/px)
	return m.el.Layout(gtx)
}
