package gui

import (
	"bytes"
	"embed"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"runtime"

	"github.com/getmiroir/trading-service/internal/svc"
)

//go:embed assets/icon.png assets/tray.png assets/icon-dev.png
var iconFS embed.FS

// The tray uses its own base image (black tile) so the status dots, green in
// particular, stay visible; the window icon matches the green app icon.
var (
	baseIcon     = mustLoadIcon(selectIcon("assets/icon.png"))
	trayBaseIcon = mustLoadIcon(selectIcon("assets/tray.png"))
)

// Icon returns the application icon bytes for use in window options.
func Icon() []byte { return baseIcon }

// pngSig is the standard PNG file signature.
var pngSig = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

// TrayBytes returns icon bytes for fyne.io/systray SetIcon. Windows Shell_NotifyIcon
// loads icons via LoadImage(IMAGE_ICON), which requires ICO; other platforms accept PNG.
func TrayBytes(pngBytes []byte) []byte {
	if runtime.GOOS != "windows" {
		return pngBytes
	}
	if len(pngBytes) < len(pngSig) || !bytes.Equal(pngBytes[:len(pngSig)], pngSig) {
		return pngBytes
	}
	return embedPNGAsICO(pngBytes)
}

// embedPNGAsICO wraps PNG data in a single-entry ICO (supported since Windows Vista).
func embedPNGAsICO(png []byte) []byte {
	w, h := icoEntryDimensions(png)
	const (
		iconDirSize   = 6
		iconEntrySize = 16
		offset        = iconDirSize + iconEntrySize
	)
	out := make([]byte, offset+len(png))
	binary.LittleEndian.PutUint16(out[0:2], 0)
	binary.LittleEndian.PutUint16(out[2:4], 1)
	binary.LittleEndian.PutUint16(out[4:6], 1)
	entry := out[6:22]
	entry[0] = w
	entry[1] = h
	entry[2] = 0
	entry[3] = 0
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 0)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(png)))
	binary.LittleEndian.PutUint32(entry[12:16], offset)
	copy(out[offset:], png)
	return out
}

func icoEntryDimensions(png []byte) (width, height uint8) {
	// IHDR: first chunk after 8-byte signature.
	if len(png) < 24 {
		return 0, 0
	}
	w := int(binary.BigEndian.Uint32(png[16:20]))
	h := int(binary.BigEndian.Uint32(png[20:24]))
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	out := func(dim int) uint8 {
		if dim >= 256 {
			return 0
		}
		return uint8(dim)
	}
	return out(w), out(h)
}

var (
	iconGreen  = iconWithDot(trayBaseIcon, color.RGBA{R: 0, G: 200, B: 0, A: 255})
	iconRed    = iconWithDot(trayBaseIcon, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	iconYellow = iconWithDot(trayBaseIcon, color.RGBA{R: 255, G: 200, B: 0, A: 255})
)

// selectIcon returns prodPath, or the shared dev icon when running as a dev service.
func selectIcon(prodPath string) string {
	if os.Getenv(svc.ServiceNameEnvVar) != "" {
		return "assets/icon-dev.png"
	}
	return prodPath
}

func mustLoadIcon(path string) []byte {
	data, err := iconFS.ReadFile(path)
	if err != nil {
		panic("failed to load embedded icon " + path + ": " + err.Error())
	}
	return data
}

func iconWithDot(base []byte, dotColor color.Color) []byte {
	src, err := png.Decode(bytes.NewReader(base))
	if err != nil {
		return base
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	radius := bounds.Dx() / 5
	outline := radius / 8
	if outline < 2 {
		outline = 2
	}
	outerRadius := radius + outline
	cx := bounds.Max.X - outerRadius - 2
	cy := bounds.Max.Y - outerRadius - 2
	rInner2 := radius * radius
	rOuter2 := outerRadius * outerRadius
	blackOutline := color.RGBA{A: 255}

	for y := cy - outerRadius; y <= cy+outerRadius; y++ {
		for x := cx - outerRadius; x <= cx+outerRadius; x++ {
			if !image.Pt(x, y).In(bounds) {
				continue
			}
			dx := x - cx
			dy := y - cy
			d2 := dx*dx + dy*dy
			if d2 <= rInner2 {
				dst.Set(x, y, dotColor)
				continue
			}
			if d2 <= rOuter2 {
				dst.Set(x, y, blackOutline)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, dst)
	return buf.Bytes()
}
