package ui

import (
	"fmt"
	"testing"
)

func TestSplitMedia(t *testing.T) {
	src := "Here is the chart:\n\n![Sales](chart.png)\n\nAnd a clip [demo](https://x.io/a.mp4) inline.\n\n" +
		"[Listen](https://x.io/talk.mp3)\n\n```\n![not](in-code.png)\n```\n[a link](https://example.com)\n"
	var got []string
	for _, p := range splitMedia(src) {
		if p.kind == "" {
			got = append(got, fmt.Sprintf("text:%q", p.text))
		} else {
			got = append(got, p.kind+":"+p.src+":"+p.alt)
		}
	}
	want := []string{
		`text:"Here is the chart:\n\n"`,
		"image:chart.png:Sales",
		`text:"\nAnd a clip [demo](https://x.io/a.mp4) inline.\n\n"`,
		"audio:https://x.io/talk.mp3:Listen",
		"text:\"\\n```\\n![not](in-code.png)\\n```\\n[a link](https://example.com)\\n\"",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestMediaPathStaysInside(t *testing.T) {
	p, err := mediaPath("../../etc/passwd", "/w")
	if err != nil || p != "/w/etc/passwd" {
		t.Fatal(p, err)
	}
}
