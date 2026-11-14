package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
	"github.com/xmhrnx/giode"
)

const (
	maxImageBytes = 25 << 20
	// maxImageSide bounds a decoded image's longer side, keeping a few
	// photos from filling the phone's memory.
	maxImageSide = 2048
	// imageCacheSize is how many decoded images stay in memory.
	imageCacheSize = 24
)

// loadedImage is an image being fetched, fetched or failed.
type loadedImage struct {
	img  image.Image
	file string // where its bytes are, for sharing
	err  error
	done bool
}

// imageCache holds decoded images by source, the least recently used
// dropped first.
type imageCache struct {
	mu    sync.Mutex
	items map[string]*loadedImage
	order []string
}

// image returns the image at src - an http(s) URL, or a path inside dir
// - starting to load it the first time; done is false until it loaded
// or failed.
func (a *App) image(src, dir string) *loadedImage {
	c := &a.images
	key := dir + "\x00" + src
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = map[string]*loadedImage{}
	}
	if it, ok := c.items[key]; ok {
		return it
	}
	it := &loadedImage{}
	c.items[key] = it
	c.order = append(c.order, key)
	if len(c.order) > imageCacheSize {
		delete(c.items, c.order[0])
		c.order = c.order[1:]
	}
	go func() {
		img, file, err := a.fetchImage(src, dir)
		c.mu.Lock()
		it.img, it.file, it.err, it.done = img, file, err, true
		c.mu.Unlock()
		a.Post(func() {})
	}()
	return it
}

// state is the image's load state for a row's change token.
func (it *loadedImage) state(c *imageCache) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case !it.done:
		return "loading"
	case it.err != nil:
		return "error"
	}
	return "ok"
}

// mediaPath resolves src inside dir, refusing paths that leave it.
func mediaPath(src, dir string) (string, error) {
	if dir == "" {
		return "", errors.New("no workspace")
	}
	rel := path.Clean("/" + strings.TrimPrefix(src, "file://"))[1:]
	if rel == "" {
		return "", errors.New("no file")
	}
	return filepath.Join(dir, filepath.FromSlash(rel)), nil
}

// fetchImage reads and decodes src, downloading a URL into the cache
// folder so it can be shared and survives restarts.
func (a *App) fetchImage(src, dir string) (image.Image, string, error) {
	var file string
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		sum := sha256.Sum256([]byte(src))
		cache := filepath.Join(a.st.Dir(), "cache", "images")
		_ = os.MkdirAll(cache, 0o755)
		file = filepath.Join(cache, hex.EncodeToString(sum[:12])+path.Ext(strings.SplitN(src, "?", 2)[0]))
		if _, err := os.Stat(file); err != nil {
			if err := download(src, file); err != nil {
				return nil, "", err
			}
		}
	} else {
		p, err := mediaPath(src, dir)
		if err != nil {
			return nil, "", err
		}
		file = p
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, "", err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", fmt.Errorf("not an image this app can show: %w", err)
	}
	return shrink(img, maxImageSide), file, nil
}

func download(src, file string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return err
	}
	if len(b) > maxImageBytes {
		return errors.New("the image is larger than 25 MB")
	}
	tmp := file + ".part"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// shrink scales img down so its longer side is at most side pixels.
func shrink(img image.Image, side int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= side && h <= side {
		return img
	}
	if w >= h {
		h, w = h*side/w, side
	} else {
		w, h = w*side/h, side
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// chatImage shows an image from a reply, fitted to the column; a tap
// opens it full screen.
func (a *App) chatImage(key, src, alt, dir string) widgets.El {
	it := a.image(src, dir)
	a.images.mu.Lock()
	img, err, done := it.img, it.err, it.done
	a.images.mu.Unlock()
	frame := widgets.St{Width: -1, MaxHeight: 420, BorderRadius: 12, Background: theme.T.Surface, BorderWidth: 1, BorderColor: theme.T.Border}
	switch {
	case !done:
		frame.Height, frame.Justify, frame.Align = 160, giode.JustifyCenter, giode.AlignCenter
		return giode.Box(frame, giode.Spinner(widgets.St{Width: 20, Height: 20, Color: theme.T.Muted}))
	case err != nil:
		return widgets.Row(widgets.St{Gap: 8, Padding: giode.UniformInset(12), BorderRadius: 12, Background: theme.T.Surface, Width: -1},
			widgets.Icon("image", 16, theme.T.Muted),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(firstNonEmpty(alt, lastPathElem(src))+" - "+tr("couldn't load the image"), theme.SizeFootnote, theme.T.Muted)))
	}
	return widgets.Tap(key, firstNonEmpty(alt, tr("Image")), func(giode.PressState) widgets.El {
		return giode.Box(widgets.St{Width: -1, MaxHeight: 420, Justify: giode.JustifyStart},
			giode.Image(img, giode.Styles{BorderRadius: 12, Fit: giode.FitContain}))
	}, func() { a.openViewer(src, dir, alt) })
}

// mediaCard stands for a video or audio file: a tap hands it to an app
// that plays it.
func (a *App) mediaCard(key, kind, src, alt, dir string) widgets.El {
	icon, label := "play", tr("Video")
	if kind == "audio" {
		icon, label = "music", tr("Audio")
	}
	title := firstNonEmpty(alt, lastPathElem(strings.SplitN(src, "?", 2)[0]))
	return widgets.ListRow(key, icon, theme.T.Primary, title, label+" · "+tr("Opens in another app"), widgets.Icon("external-link", 16, theme.T.Faint), func() {
		a.openMedia(src, dir)
	})
}

// openMedia plays video or audio in another app: a link opens in the
// system's handler for it, a workspace file in the app the system picks
// for its type.
func (a *App) openMedia(src, dir string) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		a.openLink(src)
		return
	}
	p, err := mediaPath(src, dir)
	if err != nil {
		a.toast(giode.ToastError, tr("This file no longer exists."))
		return
	}
	a.openFile(p)
}

// openFile opens a file in the app the system picks for its type.
func (a *App) openFile(path string) {
	go func() {
		if err := a.win.OpenWith(context.Background(), path); err != nil && !errors.Is(err, giode.ErrCanceled) {
			a.toast(giode.ToastError, tr("Couldn't open it: {error}", "error", err))
		}
	}()
}

// openViewer shows an image full screen, zoomable.
func (a *App) openViewer(src, dir, alt string) {
	a.viewer = viewerState{src: src, dir: dir, alt: alt}
	a.viewerZoom = nil
	a.navigate("viewer")
}

// viewerState is the image the viewer shows.
type viewerState struct{ src, dir, alt string }

func (a *App) imageViewer(compact bool) widgets.El {
	v := a.viewer
	it := a.image(v.src, v.dir)
	a.images.mu.Lock()
	img, file := it.img, it.file
	a.images.mu.Unlock()
	actions := widgets.Row(widgets.St{Gap: 2},
		widgets.When(file != "", func() widgets.El {
			return widgets.IconButton("viewer/share", "share", tr("Share"), 18, theme.T.Text2, func() { a.shareFile(file, mimeOf(file)) })
		}))
	return a.pageScaffold("viewer", firstNonEmpty(v.alt, tr("Image")), compact, actions, func() widgets.El {
		if img == nil {
			return giode.Box(widgets.St{Width: -1, Height: 200, Justify: giode.JustifyCenter, Align: giode.AlignCenter},
				giode.Spinner(widgets.St{Width: 22, Height: 22, Color: theme.T.Muted}))
		}
		if a.viewerZoom == nil {
			a.viewerZoom = giode.Zoomable(func() giode.Element {
				return giode.Image(img, giode.Styles{Width: -1})
			}).Limits(1, 8)
		}
		return giode.Box(widgets.St{Width: -1, Height: 560}, a.viewerZoom)
	})
}
