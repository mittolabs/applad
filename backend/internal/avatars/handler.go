// Package avatars generates avatar images, QR placeholders, favicons,
// credit-card brand badges, and country flag SVGs on the fly.
package avatars

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler handles avatar HTTP requests.
type Handler struct{}

// NewHandler creates a new avatars Handler.
func NewHandler() *Handler { return &Handler{} }

// Routes returns the avatars router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/initials", h.initials)
	r.Get("/qr", h.qr)
	r.Get("/favicon", h.favicon)
	r.Get("/credit-cards/{code}", h.creditCard)
	r.Get("/flags/{code}", h.flag)
	r.Get("/browsers/{code}", h.browser)
	r.Get("/image", h.remoteImage)
	return r
}

// ---------------------------------------------------------------------------
// GET /initials?name=John+Doe&width=100&height=100&background=6C47FF
// ---------------------------------------------------------------------------

// 5x7 pixel font for uppercase A-Z and digits 0-9.
var glyphs = map[rune][7]uint8{
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C': {0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G': {0b01110, 0b10001, 0b10000, 0b10111, 0b10001, 0b10001, 0b01110},
	'H': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'I': {0b01110, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'J': {0b00111, 0b00010, 0b00010, 0b00010, 0b00010, 0b10010, 0b01100},
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'O': {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S': {0b01110, 0b10001, 0b10000, 0b01110, 0b00001, 0b10001, 0b01110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b11011, 0b10001},
	'X': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
	'0': {0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'2': {0b01110, 0b10001, 0b00001, 0b00110, 0b01000, 0b10000, 0b11111},
	'3': {0b01110, 0b10001, 0b00001, 0b00110, 0b00001, 0b10001, 0b01110},
	'4': {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5': {0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110},
	'6': {0b01110, 0b10000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00001, 0b01110},
}

func (h *Handler) initials(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "?"
	}

	width := queryInt(r, "width", 100)
	height := queryInt(r, "height", 100)
	bgHex := r.URL.Query().Get("background")
	if bgHex == "" {
		bgHex = "6C47FF"
	}

	bg := parseHexColor(bgHex)
	fg := color.White

	// Extract initials (first letter of each word, max 2).
	words := strings.Fields(name)
	var initials []rune
	for _, word := range words {
		if len(initials) >= 2 {
			break
		}
		for _, ch := range word {
			initials = append(initials, ch)
			break
		}
	}
	if len(initials) == 0 {
		initials = []rune{'?'}
	}

	// Uppercase the initials.
	text := strings.ToUpper(string(initials))

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Determine scale: each glyph is 5 wide, 7 tall, 1px gap between glyphs.
	glyphCount := len([]rune(text))
	totalGlyphW := glyphCount*5 + (glyphCount-1)*1 // with 1px gaps
	totalGlyphH := 7

	// Scale to fit roughly 60% of the image dimension.
	maxW := float64(width) * 0.6
	maxH := float64(height) * 0.6
	scaleX := maxW / float64(totalGlyphW)
	scaleY := maxH / float64(totalGlyphH)
	scale := int(math.Min(scaleX, scaleY))
	if scale < 1 {
		scale = 1
	}

	// Compute the actual rendered size after scaling.
	renderedW := totalGlyphW * scale
	renderedH := totalGlyphH * scale

	// Top-left offset to center.
	offsetX := (width - renderedW) / 2
	offsetY := (height - renderedH) / 2

	// Draw each glyph.
	cursorX := offsetX
	for _, ch := range text {
		glyph, ok := glyphs[ch]
		if !ok {
			// Unknown character — draw a solid block.
			for row := 0; row < 7; row++ {
				glyph[row] = 0b11111
			}
		}
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row]&(1<<uint(4-col)) != 0 {
					// Fill a scale×scale block.
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							px := cursorX + col*scale + dx
							py := offsetY + row*scale + dy
							if px >= 0 && px < width && py >= 0 && py < height {
								img.Set(px, py, fg)
							}
						}
					}
				}
			}
		}
		cursorX += (5 + 1) * scale // 5 glyph width + 1 gap
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	png.Encode(w, img)
}

// ---------------------------------------------------------------------------
// GET /qr?text=hello&size=200
// ---------------------------------------------------------------------------

func (h *Handler) qr(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" {
		text = "https://applad.io"
	}
	size := queryInt(r, "size", 200)

	// Generate a deterministic visual pattern from the text hash.
	hash := md5.Sum([]byte(text))

	// Build an SVG with a grid derived from the hash, plus the text embedded
	// as a title element for accessibility.
	gridSize := 8
	cellSize := size / (gridSize + 2) // leave 1-cell quiet zone
	if cellSize < 1 {
		cellSize = 1
	}
	totalPx := cellSize * (gridSize + 2)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, size, size, totalPx, totalPx)
	fmt.Fprintf(&b, `<title>%s</title>`, xmlEscape(text))
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="white"/>`, totalPx, totalPx)

	// Draw finder patterns (three corners).
	drawFinderPattern(&b, cellSize, cellSize, cellSize)
	drawFinderPattern(&b, (gridSize-6)*cellSize, cellSize, cellSize)
	drawFinderPattern(&b, cellSize, (gridSize-6)*cellSize, cellSize)

	// Fill data cells from hash bytes.
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			byteIdx := (row*gridSize + col) % len(hash)
			bitIdx := (row*gridSize + col) % 8
			if hash[byteIdx]&(1<<uint(bitIdx)) != 0 {
				x := (col + 1) * cellSize
				y := (row + 1) * cellSize
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`, x, y, cellSize, cellSize)
			}
		}
	}

	b.WriteString(`</svg>`)

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(b.String()))
}

func drawFinderPattern(b *strings.Builder, x, y, cell int) {
	// 7x7 finder: outer black, inner white, center black (3x3).
	for row := 0; row < 7; row++ {
		for col := 0; col < 7; col++ {
			isOuter := row == 0 || row == 6 || col == 0 || col == 6
			isInner := row >= 2 && row <= 4 && col >= 2 && col <= 4
			if isOuter || isInner {
				fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`,
					x+col*cell, y+row*cell, cell, cell)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// GET /favicon?url=example.com
// ---------------------------------------------------------------------------

func (h *Handler) favicon(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, `{"message":"Missing url parameter","code":400,"type":"general_argument_invalid"}`, http.StatusBadRequest)
		return
	}

	// Normalise: ensure scheme is present.
	target := rawURL
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}
	faviconURL := strings.TrimRight(target, "/") + "/favicon.ico"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(faviconURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Return a default 1x1 transparent PNG.
		writeDefaultFavicon(w)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/x-icon"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, resp.Body)
}

func writeDefaultFavicon(w http.ResponseWriter) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{108, 71, 255, 255}}, image.Point{}, draw.Src)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	png.Encode(w, img)
}

// ---------------------------------------------------------------------------
// GET /credit-cards/{code}  — visa, mastercard, amex, discover, diners, jcb, unionpay
// ---------------------------------------------------------------------------

var cardColors = map[string]string{
	"visa":       "#1A1F71",
	"mastercard": "#EB001B",
	"amex":       "#2E77BC",
	"discover":   "#FF6600",
	"diners":     "#0079BE",
	"jcb":        "#0B7BC0",
	"unionpay":   "#D40000",
}

var cardLabels = map[string]string{
	"visa":       "VISA",
	"mastercard": "Mastercard",
	"amex":       "AMEX",
	"discover":   "Discover",
	"diners":     "Diners Club",
	"jcb":        "JCB",
	"unionpay":   "UnionPay",
}

func (h *Handler) creditCard(w http.ResponseWriter, r *http.Request) {
	code := strings.ToLower(chi.URLParam(r, "code"))

	bgColor, ok := cardColors[code]
	if !ok {
		http.Error(w, `{"message":"Unknown card code","code":404,"type":"general_not_found"}`, http.StatusNotFound)
		return
	}
	label := cardLabels[code]

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">
  <rect width="120" height="80" rx="8" fill="%s"/>
  <text x="60" y="46" font-family="Arial,Helvetica,sans-serif" font-size="14" font-weight="bold" fill="white" text-anchor="middle">%s</text>
</svg>`, bgColor, xmlEscape(label))

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write([]byte(svg))
}

// ---------------------------------------------------------------------------
// GET /flags/{code}  — 2-letter country code
// ---------------------------------------------------------------------------

func (h *Handler) flag(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	if len(code) != 2 {
		http.Error(w, `{"message":"Invalid country code","code":400,"type":"general_argument_invalid"}`, http.StatusBadRequest)
		return
	}

	// Convert two ASCII letters to regional indicator emoji pair.
	// 'A' (0x41) maps to U+1F1E6, 'B' to U+1F1E7, etc.
	r1 := rune(0x1F1E6 + (rune(code[0]) - 'A'))
	r2 := rune(0x1F1E6 + (rune(code[1]) - 'A'))
	emoji := string([]rune{r1, r2})

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">
  <rect width="120" height="80" rx="4" fill="#F0F0F0"/>
  <text x="60" y="56" font-size="48" text-anchor="middle">%s</text>
</svg>`, emoji)

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write([]byte(svg))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.RGBA{108, 71, 255, 255} // default brand purple
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// ---------------------------------------------------------------------------
// GET /browsers/{code} — browser icon SVG by browser name
// ---------------------------------------------------------------------------

var browserColors = map[string]string{
	"chrome": "#4285F4", "firefox": "#FF7139", "safari": "#006CFF",
	"edge": "#0078D7", "opera": "#FF1B2D", "brave": "#FB542B",
	"vivaldi": "#EF3939", "arc": "#0095FF", "tor": "#7D4698",
	"samsung": "#1428A0", "ie": "#0076D6", "unknown": "#888888",
}

var browserLabels = map[string]string{
	"chrome": "Ch", "firefox": "Fx", "safari": "Sa", "edge": "Ed",
	"opera": "Op", "brave": "Br", "vivaldi": "Vi", "arc": "Ar",
	"tor": "To", "samsung": "Si", "ie": "IE", "unknown": "?",
}

func (h *Handler) browser(w http.ResponseWriter, r *http.Request) {
	code := strings.ToLower(chi.URLParam(r, "code"))
	width := queryInt(r, "width", 48)
	height := queryInt(r, "height", 48)

	bgColor, ok := browserColors[code]
	if !ok {
		bgColor = browserColors["unknown"]
	}
	label, ok := browserLabels[code]
	if !ok {
		label = strings.ToUpper(code[:min(2, len(code))])
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
  <rect width="%d" height="%d" rx="8" fill="%s"/>
  <text x="50%%" y="54%%" text-anchor="middle" dominant-baseline="middle" font-family="system-ui,-apple-system,sans-serif" font-size="%d" font-weight="700" fill="white">%s</text>
</svg>`, width, height, width, height, width, height, bgColor, width/3, xmlEscape(label))

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.WriteString(w, svg)
}

// ---------------------------------------------------------------------------
// GET /image?url=...&width=...&height=...&output=png — fetch and transform remote image
// ---------------------------------------------------------------------------

func (h *Handler) remoteImage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	// Validate URL starts with http
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		http.Error(w, "url must start with http:// or https://", http.StatusBadRequest)
		return
	}

	width := queryInt(r, "width", 0)
	height := queryInt(r, "height", 0)

	// Fetch remote image
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		http.Error(w, "failed to fetch image", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		http.Error(w, fmt.Sprintf("remote server returned %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	// Limit to 10MB
	body := io.LimitReader(resp.Body, 10<<20)

	// If no resize needed, proxy through
	if width == 0 && height == 0 {
		ct := resp.Header.Get("Content-Type")
		if ct == "" {
			ct = "image/png"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		io.Copy(w, body)
		return
	}

	// Decode, resize, and serve
	src, _, err := image.Decode(body)
	if err != nil {
		http.Error(w, "failed to decode image", http.StatusBadRequest)
		return
	}

	srcBounds := src.Bounds()
	if width == 0 {
		width = srcBounds.Dx() * height / srcBounds.Dy()
	}
	if height == 0 {
		height = srcBounds.Dy() * width / srcBounds.Dx()
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// Simple nearest-neighbor resize
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * srcBounds.Dx() / width
			srcY := y * srcBounds.Dy() / height
			dst.Set(x, y, src.At(srcX+srcBounds.Min.X, srcY+srcBounds.Min.Y))
		}
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	png.Encode(w, dst)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

