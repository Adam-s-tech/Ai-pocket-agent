package ui

import (
	"path"
	"regexp"
	"strings"
)

// mediaPart is a piece of a reply: Markdown text, or an image, video or
// audio file that stood on a line of its own.
type mediaPart struct {
	text string
	// kind is "", "image", "video" or "audio"; src and alt describe the
	// media.
	kind, src, alt string
}

// mediaLine matches a line holding only Markdown images or links:
// ![alt](src) or [alt](src), optionally with a title.
var mediaLine = regexp.MustCompile(`^\s*(?:!?\[[^\]]*\]\(\s*<?[^)\s>]+>?(?:\s+"[^"]*")?\s*\)\s*)+$`)

var mediaItem = regexp.MustCompile(`(!?)\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)

// mediaKind classifies a file or URL by its extension.
func mediaKind(src string) string {
	p := src
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv", ".avi", ".3gp":
		return "video"
	case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".oga", ".opus", ".flac":
		return "audio"
	}
	return ""
}

// splitMedia splits Markdown at lines that hold only images, or links
// to video or audio, outside code blocks, so they can be shown as media
// rather than text. Markdown images whose address has no image
// extension still count as images.
func splitMedia(s string) []mediaPart {
	if !strings.Contains(s, "](") {
		return []mediaPart{{text: s}}
	}
	var parts []mediaPart
	var text strings.Builder
	flush := func() {
		if strings.TrimSpace(text.String()) != "" {
			parts = append(parts, mediaPart{text: text.String()})
		}
		text.Reset()
	}
	fence := ""
	for _, line := range strings.SplitAfter(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			switch {
			case fence == "":
				fence = t[:3]
			case strings.HasPrefix(t, fence):
				fence = ""
			}
		}
		if fence != "" || !mediaLine.MatchString(t) {
			text.WriteString(line)
			continue
		}
		var media []mediaPart
		for _, m := range mediaItem.FindAllStringSubmatch(t, -1) {
			kind := mediaKind(m[3])
			if m[1] == "!" && kind == "" {
				kind = "image"
			}
			if kind == "" || (m[1] != "!" && kind == "image") && !strings.HasPrefix(m[3], "http") {
				media = nil // an ordinary link: the line stays text
				break
			}
			media = append(media, mediaPart{kind: kind, src: m[3], alt: m[2]})
		}
		if len(media) == 0 {
			text.WriteString(line)
			continue
		}
		flush()
		parts = append(parts, media...)
	}
	flush()
	if len(parts) == 0 {
		return []mediaPart{{text: s}}
	}
	return parts
}
