package xgw
import (
	"time"
	"strings"
    "github.com/BurntSushi/xgb/xproto"
    "github.com/BurntSushi/xgb"
    "github.com/BurntSushi/xgbutil"
    "github.com/BurntSushi/xgb/xtest"
)
type Window = xproto.Window
type EXProp = xproto.PropertyNotifyEvent
type EXClient = xproto.ClientMessageEvent
type EXButton = xproto.ButtonPressEvent
type EXKey = xproto.KeyPressEvent
type EXSel = xproto.SelectionRequestEvent
type EXMap = xproto.MapNotifyEvent
type EXCreate = xproto.CreateNotifyEvent
type EXDestroy = xproto.DestroyNotifyEvent
type EXUnmap = xproto.UnmapNotifyEvent
type windowState struct { mapped bool; barData string }
var (
    conn *xgb.Conn
	xu *xgbutil.XUtil
	screen *xproto.ScreenInfo
	height, width, scale, deskID int
	clipboard, currentDesktop string
	timeHour = time.Hour
	atomMap = make(map[string]xproto.Atom)
	WinStates = make(map[Window]windowState)
    DesktopWins, StickyWins []Window
    BarWindow, ImWindow, root, FocusWindow Window
	timeDiff, selTime, deskLock uint32
)
func QueryTree(win Window, callback func (Window)) { if tree, err := xproto.QueryTree(conn, win).Reply(); err == nil { for _, sub := range tree.Children { callback(sub) } } }
func QueryBytes(win Window, prop string) (ret []byte) { if reply, err := xproto.GetProperty(conn, false, win, atomMap[prop], atomMap["STRING"], 0, 1024).Reply(); err == nil { ret = reply.Value }; return }
func SendString(win Window, prop xproto.Atom, str string) { SendBytes(win, prop, atomMap["STRING"], 8, []byte(str)) }
func SendBytes(win Window, prop, propType xproto.Atom, format byte, data []byte) { xproto.ChangeProperty(conn, xproto.PropModeReplace, win, prop, propType, format, uint32(len(data)*8 / int(format)), data) }
func FocusSet(win Window) { if win != FocusWindow && win != 0 { xproto.SetInputFocus(conn, xproto.InputFocusPointerRoot, win, 0); FocusWindow = win } }
func RaiseWindow(win Window) { xproto.ConfigureWindow(conn, win, xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove}) }
func GetGeometry(win Window) (x,y, w, h int) { if reply, err := xproto.GetGeometry(conn, xproto.Drawable(win)).Reply(); err == nil { return int(reply.X), int(reply.Y), int(reply.Width), int(reply.Height) }; return }
func Map(win Window) bool { return xproto.MapWindow(conn, win).Check() == nil }
func Unmap(win Window) bool { return xproto.UnmapWindow(conn, win).Check() == nil }
func QueryPointer() (int, int) { reply, _ := xproto.QueryPointer(conn, root).Reply(); return int(reply.RootX), int(reply.RootY) }
func SetClipboard(selName, text string) { selTime, clipboard = XTimeNow(), text; xproto.SetSelectionOwner(conn, BarWindow, atomMap[selName], xproto.Timestamp(selTime)) }
func ResizeWindow(win Window, x, y, w, h int) { xproto.ConfigureWindow(conn, win, uint16(xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth | xproto.ConfigWindowHeight | xproto.ConfigWindowBorderWidth), []uint32{Abs32(x), Abs32(y), Abs32(w), Abs32(h), 0}) }
func GetWindowPID(win Window) (ret uint32) { if reply, err := xproto.GetProperty(conn, false, win, atomMap["_NET_WM_PID"], atomMap["CARDINAL"], 0, 1).Reply(); err == nil && len(reply.Value) >= 4 { ret = *Ptr[uint32](&reply.Value[0]) }; return }
func GetTitle(win Window) string { return string(QueryBytes(win, "WM_NAME")) }
func SetWmName(win Window, name string) { SendString(win, atomMap["WM_NAME"], name) }
func XTimeNow() uint32 { return timeDiff+uint32(time.Now().UnixMilli()) }
func SetXTime(t uint32) { timeDiff = t-uint32(time.Now().UnixMilli()) }
func FindWindow(title string) Window { for win, _ := range WinStates { if strings.Contains(GetTitle(win), title) { return win } }; return 0 }
func CountWindowsOfTitle(title string) (count int) { for _, state := range WinStates { if strings.Contains(state.barData, title) { count +=1; continue } }; return }

func init() {
	initFont()
    var err error
	conn, err = xgb.NewConn()
	logAndExit(err)
    screen = xproto.Setup(conn).DefaultScreen(conn)
    root, FocusWindow, height, width = screen.Root, screen.Root, int(screen.HeightInPixels), int(screen.WidthInPixels)
    if width < 1000 { scale = 1 } else { scale = int(width/1000) }
    for _, atomName := range xconf.X11Atoms{ setupAtom(atomName, true) }
	setupAtom(xconf.BarAtom, false)
    xu, err = xgbutil.NewConn()
	logAndExit(err, xproto.ChangeWindowAttributesChecked(conn, root, xproto.CwEventMask, []uint32{uint32(xproto.EventMaskSubstructureNotify)}).Check())
	QueryTree(root, syncState)
}

func syncState(win Window) {
	if _, exists := WinStates[win]; exists { return }
    attr, _ := xproto.GetWindowAttributes(conn, win).Reply()
	WinStates[win] = windowState{mapped: attr.MapState == xproto.MapStateViewable, barData: string(QueryBytes(win, xconf.BarAtom))}
}

func setupAtom(atomName string, existingAtom bool) {
    atom, err := xproto.InternAtom(conn, existingAtom, uint16(len(atomName)), atomName).Reply()
    logAndExit(err)
    atomMap[atomName] = atom.Atom
}

func EmulateSequence(keys ...string) {
	conn, err := xgb.NewConn()
	if err != nil { return }
	defer conn.Close()
	if time.Sleep(time.Second/4); xtest.Init(conn) != nil { return }	
	for i := 0; i < len(keys); i++ { xtest.FakeInput(conn, xproto.KeyPress, byte(parseInt(keys[i])), 0, 0, 0, 0, 0) }
	for i := len(keys)-1; i>=0; i-- { xtest.FakeInput(conn, xproto.KeyRelease, byte(parseInt(keys[i])), 0, 0, 0, 0, 0) }
} 

type XImage struct { Conn *xgb.Conn; Pixmap xproto.Pixmap; Win Window; Width, Height int }
func (im *XImage) Flush() { xproto.ClearArea(xu.Conn(), false, im.Win, 0, 0, 0, 0); im.Conn.Sync() }
func (im *XImage) Ungrab(code byte) { xproto.UngrabKey(im.Conn, xproto.Keycode(code), root, xproto.ModMaskAny) }
func (im *XImage) Grab(mod uint16, code byte) { xproto.GrabKey(im.Conn, false, root, mod, xproto.Keycode(code), xproto.GrabModeAsync, xproto.GrabModeAsync) }

func NewXImage(x, y, w, h int, title string) (ret *XImage) { 
	var err error
	ret = &XImage{ Width: w, Height: h, Pixmap: 0, Win: root, Conn: conn }
	if title != "root" {
		if title != barTitle { ret.Conn, err = xgb.NewConn(); logAndExit(err) }
		if ret.Win, err = xproto.NewWindowId(ret.Conn); err != nil || xproto.CreateWindowChecked(
			ret.Conn, screen.RootDepth, ret.Win, root, int16(x), int16(y), uint16(w), uint16(h), 0, // border width
			xproto.WindowClassInputOutput, screen.RootVisual, xproto.CwBackPixel|xproto.CwEventMask,
			[]uint32{ uint32(screen.BlackPixel), xproto.EventMaskKeyPress | xproto.EventMaskStructureNotify | xproto.EventMaskPropertyChange | xproto.EventMaskButtonPress},
		).Check() != nil { ret.Destroy(); return nil }
		setWmName(ret.Win, title)
		hintsWindow := atomMap["WM_DELETE_WINDOW"]
		SendBytes(ret.Win, atomMap["WM_PROTOCOLS"], atomMap["ATOM"], 32, Array[byte](&hintsWindow, 4))
		Map(ret.Win)
	}
	if ret.Pixmap, err = xproto.NewPixmapId(xu.Conn()); err != nil { ret.Destroy(); return nil }
	if logErr(xproto.CreatePixmapChecked(xu.Conn(), screen.RootDepth, ret.Pixmap, xproto.Drawable(xu.RootWin()), uint16(w), uint16(h)).Check()) { ret.Destroy(); return nil }
	xproto.ChangeWindowAttributes(xu.Conn(), ret.Win, xproto.CwBackPixmap, []uint32{uint32(ret.Pixmap)})
	return ret
}

func (im *XImage) Destroy() { 
	if im == nil { return }
	if im.Pixmap != 0 { xproto.FreePixmap(xu.Conn(), im.Pixmap); im.Pixmap = 0 } 	
	if im.Win != root && im.Win != 0 { xproto.DestroyWindow(im.Conn, im.Win); im.Win = 0 }
	if im.Conn != conn && im.Conn != nil { im.Conn.Close(); im.Conn = nil }
}

func (im *XImage) XDraw(img RGBAData, xpos, ypos int) {
	var data, toSend []uint8
	if (img.Stride == img.Width*4) {
		data = Array[uint8](&img.Pix[0], img.Height*img.Stride)
	} else {
		data = make([]uint8, 0, 4*img.Width*img.Height)[:4*img.Width*img.Height]
		for i := 0; i < img.Height ; i++ {
			pos, dest := i * img.Stride, i * img.Width * 4
			copy(data[dest:dest+4*img.Width], Array[uint8](&img.Pix[0], img.Height*img.Stride)[pos:pos+4*img.Width])
		}
	}
	rowsPer := (xgbutil.MaxReqSize - 28) / (img.Width * 4) // X's max request size (by default) is (2^16) * 4 = 262144 bytes, which corresponds precisely to a 256x256 sized image with 32 bits per pixel. The constant 28 comes from the fixed size part of a PutImage request.
	bytesPer, start, end := rowsPer*img.Width*4, 0, 0
	for end < len(data) {
		end = start + bytesPer
		if end > len(data) { end = len(data) }
		toSend = data[start:end]
		xproto.PutImage(xu.Conn(), xproto.ImageFormatZPixmap, xproto.Drawable(im.Pixmap), xu.GC(), uint16(img.Width), uint16(len(toSend)/4/img.Width), int16(xpos), int16(ypos), 0, 24, toSend)
		start = end
		ypos += rowsPer
	}
}

func UseClipboard(client Window, clientProp, target, selection xproto.Atom, timeStamp xproto.Timestamp) {
	if clientProp == atomMap["NONE"] { clientProp = target }
	var propType xproto.Atom
	var propFormat byte = 32
	var data []byte
	var data32 [2]uint32
	switch target {
	case atomMap["TARGETS"]: propType, data, data32[0], data32[1] = atomMap["ATOM"], Array[byte](&data32[0], 8), uint32(atomMap["TARGETS"]), uint32(atomMap["UTF8_STRING"])
	case atomMap["TIMESTAMP"]: propType, data, data32[0] = atomMap["INTEGER"], Array[byte](&data32[0], 4), uint32(selTime)
	default: propType, propFormat, data = atomMap["UTF8_STRING"], 8, []byte(clipboard)
	}
	SendBytes(client, clientProp, propType, propFormat, data)
	xproto.SendEvent(conn, false, client, xproto.EventMaskNoEvent, string(xproto.SelectionNotifyEvent{Time: timeStamp, Requestor: client, Selection: selection, Target: target, Property: clientProp}.Bytes()))
}

func SendWmDelete(win Window) {
    data32 := [5]uint32 { uint32(atomMap["WM_DELETE_WINDOW"]), 0, 0, 0, 0 }
    xproto.SendEvent(conn, false, win, xproto.EventMaskNoEvent, string(xproto.ClientMessageEvent{Format: 32, Window: win, Type: atomMap["WM_PROTOCOLS"], Data: xproto.ClientMessageDataUnion{Data8: Array[byte](&data32[0], 20)}}.Bytes())) // Only Data8 works
}

func Screenshot(x, y, w, h int) ([]byte, []uint32) {
	if reply, err := xproto.GetImage(conn, xproto.ImageFormatZPixmap, xproto.Drawable(root), int16(x), int16(y), uint16(w), uint16(h), 0xFFFFFFFF).Reply(); err == nil { return reply.Data, Array[uint32](&reply.Data[0], w*h) }
	return nil, nil
}
