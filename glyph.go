package xgw
/*
#cgo LDFLAGS: -L/output/static-libs -l:libfreetype.a -lm
#cgo CFLAGS:  -I/output/include/freetype -I/output/include/
#include "CPlugins/src/plugin-ff2.c"
#include "CPlugins/src/plugin-missing.c"
*/
import "C"
import (
	"unsafe"
	"log"
	"errors"
	"os"
	_ "embed"
	"encoding/json"
	"bytes"
)
const (
    GlyphWidth, GlyphHeight, glyphBaseline = 24, 40, 34
	BarTitle = "auto-stickybar"
)
type RGBAData struct { Pix []uint32; Width, Height, Stride int }
type cStr struct { data []byte; Ptr *C.char }
type X11Config struct {
	Keymap [190]string `json:"x11_keymap"`
	XTermColors [16]uint32 `json:"xterm_colors"`
	X11Atoms []string `json:"x11_atoms"`
	BarAtom string `json:"bar_atom"`
	Fonts [3]string `json:"fonts"`
}
var (
	glyphAtlas []uint32
	coloredGlyphs = make(map[uint64]int)
	Conf X11Config
	//go:embed x11.json
	ConfData []byte
	ErrXImg = errors.New("XImage error")
	ErrUnknown = errors.New("unknown error")
	ErrEmpty = errors.New("empty error")
	ErrFull = errors.New("full error")
	ErrNotFound = errors.New("not found err")
	ErrLoad = errors.New("failed to load")
	ErrLib = errors.New(".so error")
	ErrResize = errors.New("failed to resize image")
	ErrParam = errors.New("invalid parameters")
	ff2Flag bool 
)
func Ptr[T any, U any](b *U) *T { return (*T)(unsafe.Pointer(b)) }
func Array[T any, U any](b *U, size int) []T { return (*(*[1<<30]T)(unsafe.Pointer(b)))[:size:size] }
func CStr(str string) (ret cStr) { ret.data = CStrBytes(str); ret.Ptr = Ptr[C.char](&ret.data[0]); return }
func BlankImage(w, h int) RGBAData { return RGBAData {Pix: make([]uint32, w*h), Stride: w*4, Width: w, Height: h} }
func Crop(img RGBAData, x0, y0, w, h int) RGBAData { return RGBAData {Pix: img.Pix[(img.Stride/4)*y0+x0:], Stride: img.Stride, Width: w, Height: h} }
func logErr(err error) bool { if err != nil { log.Printf("Err: %v", err) }; return err != nil }

func LogAndExit(cleanup func(), errs ...error) { 
	for _, err := range errs { 
		if err == nil { continue }
		log.Printf("Fatal: %v", err); 
		if cleanup != nil { cleanup() }
		Cleanup()
		os.Exit(1) 
	} 
}

func Cleanup() {
	if ff2Flag { C.ft_cleanup(); ff2Flag = false }
	if xu != nil { xu.Conn().Close(); xu = nil }
	if conn != nil { conn.Close(); conn = nil }
}

func logAndExit(errs ...error) { LogAndExit(nil, errs...) }

func initFont() { 
    logAndExit(json.NewDecoder(bytes.NewReader(ConfData)).Decode(&Conf))
    for i, _ := range Conf.Fonts { Conf.Fonts[i] = ExpandHome(Conf.Fonts[i]) }
	C.ft_init(CStr(Conf.Fonts[0]).Ptr, CStr(Conf.Fonts[1]).Ptr, CStr(Conf.Fonts[2]).Ptr, C.int(GlyphHeight)); ff2Flag = true 
}

func GetColoredGlyph(aRune, fgColor, bgColor uint32) RGBAData {
    cacheKey := uint64(aRune) | (uint64(fgColor*1007+bgColor)) << 32
	if ret, exists := coloredGlyphs[cacheKey]; exists { 
		tWidth, offset := GlyphWidth*(1+1&ret), (ret>>1)
		return RGBAData {Pix: glyphAtlas[offset:offset+GlyphHeight*tWidth], Width: tWidth, Height: GlyphHeight, Stride: tWidth*4}
	}
	if cap(glyphAtlas) < len(glyphAtlas) + 2*GlyphWidth*GlyphHeight {
		newAtlas := make([]uint32, 0, GlyphWidth*GlyphHeight*100 + cap(glyphAtlas))[:cap(glyphAtlas)]
		copy(newAtlas, glyphAtlas)
		glyphAtlas = newAtlas
	}
	offset := len(glyphAtlas)
	glyphAtlas = glyphAtlas[:offset+2*GlyphWidth*GlyphHeight]
	tWidth := int(C.make_ff2_glyph(Ptr[C.char](&aRune), C.uint32_t(fgColor), C.uint32_t(bgColor), C.int(GlyphWidth*2), C.int(GlyphHeight), C.int(glyphBaseline), Ptr[C.uint32_t](&glyphAtlas[offset])))
	if tWidth == 0 { tWidth = GlyphWidth }
	if tWidth > GlyphWidth {
		coloredGlyphs[cacheKey] = offset<<1 + 1
	} else {
		coloredGlyphs[cacheKey] = offset<<1
		glyphAtlas = glyphAtlas[:offset+GlyphWidth*GlyphHeight]
	}
	return RGBAData {Pix: glyphAtlas[offset:offset+GlyphHeight*tWidth], Width: tWidth, Height: GlyphHeight, Stride: tWidth*4}
}
