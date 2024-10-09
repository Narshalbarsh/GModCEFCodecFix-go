package ui

import (
	_ "embed"
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

//go:embed GModPatchToolLogo.png
var BgImgData []byte

type TransparentEntry struct {
	widget.Entry
}

func NewTransparentEntry() *TransparentEntry {
	entry := &TransparentEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}
func (e *TransparentEntry) TypedRune(r rune) {
	// Do nothing
}
func (e *TransparentEntry) TypedKey(key *fyne.KeyEvent) {
	// Do nothing
}

// Custom renderer to make the textbox background transparent
func (t *TransparentEntry) CreateRenderer() fyne.WidgetRenderer {
	renderer := t.Entry.CreateRenderer()
	for _, obj := range renderer.Objects() {
		if bg, ok := obj.(*canvas.Rectangle); ok {
			bg.Hide()
		}
	}
	return renderer
}

// Custom layout to position the background image at the bottom right
type BottomRightLayout struct{}

func (b *BottomRightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		if img, ok := obj.(*canvas.Image); ok {
			img.Resize(fyne.NewSize(200, 200))
			img.Move(fyne.NewPos(size.Width-img.Size().Width-50, size.Height-img.Size().Height-50))
		} else {
			obj.Resize(size)
		}
	}
}
func (b *BottomRightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(400, 400)
}

// implements the io.Writer interface
type CustomStream struct {
	orig                      io.Writer
	textWidgetWriter          func(string)
	textWidgetPositionUpdater func()
}

func (c *CustomStream) Write(p []byte) (n int, err error) {
	c.textWidgetWriter(string(p))
	c.textWidgetPositionUpdater()
	c.orig.Write(p)
	return len(p), nil
}

// Write all of the stodut and stderr to a text widget
func InterceptTextOutputToGui(textBox *TransparentEntry) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rStdout, wStdout, _ := os.Pipe()
	rStderr, wStderr, _ := os.Pipe()
	textBoxPositionUpdater := func() {
		textBox.CursorRow = (len(textBox.Text) - 1)
	}
	stdoutStream := &CustomStream{
		orig:                      origStdout,
		textWidgetWriter:          textBox.Append,
		textWidgetPositionUpdater: textBoxPositionUpdater,
	}
	stderrStream := &CustomStream{
		orig:                      origStderr,
		textWidgetWriter:          textBox.Append,
		textWidgetPositionUpdater: textBoxPositionUpdater,
	}
	os.Stdout = wStdout
	os.Stderr = wStderr
	go io.Copy(stdoutStream, rStdout)
	go io.Copy(stderrStream, rStderr)
}
