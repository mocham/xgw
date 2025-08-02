package xgw
import "strings"
func UniversalWidget(title string, left, top, winWidth, winHeight int, paint func (*XImage) (int, int), button func (byte, int16, int16) int, keypress func (byte) int, refresh func(string), init func(*XImage)) {
	ximg := NewXImage(left, top, winWidth, winHeight, title)
	if ximg == nil { return }
	defer func() { ximg.Ungrab(0); ximg.Destroy() }()
	if init != nil { init(ximg) }
	paintWrap := func () {
		w, h := paint(ximg)
		if w >0 && h > 0 { ResizeWindow(ximg.Win, left, top, w, h) }
		ximg.Flush()
	}
	paintWrap()
	for {
        ev, err := ximg.Conn.WaitForEvent()
        if err != nil || ev == nil { continue }
        switch event := ev.(type) {
		case EXProp:
			if refresh == nil { continue }
			if newTitle := GetTitle(ximg.Win); len(newTitle)>0 && newTitle[len(newTitle)-1] == '*' {
				refresh(newTitle)
				paintWrap()
				SetWmName(ximg.Win, title)
			}
        case EXClient: if event.Type==AtomMap["WM_PROTOCOLS"] && event.Data.Data32[0]==uint32(AtomMap["WM_DELETE_WINDOW"]) && ximg.Win==event.Window { return }
        case EXButton:
			if button == nil { continue }
			switch button(byte(event.Detail), event.RootX, event.RootY) {
			case 1: paintWrap()
			case -1: return
			}
		case EXKey:
			if keypress == nil { continue }
			detail := byte(event.Detail)
			if event.State != 0 { detail += 128 }
			if int(detail) > len(Conf.Keymap) { detail = 0 }
			switch keypress(detail) {
			case 1: paintWrap()
			case -1: return
			}
		}
	}
}
func WindowRaiseFocuser(ximg *XImage) { RaiseWindow(ximg.Win); FocusSet(ximg.Win) }

// Dequeue implements a fixed-capacity double-ended queue.
type Dequeue[T any] struct { data []T; capacity, size, head, tail int }
func NewDequeue[T any](capacity int) *Dequeue[T] { return &Dequeue[T]{ data: make([]T, capacity), capacity: capacity, head: 0, tail: 0, size: 0} }

func (d *Dequeue[T]) PushFront(val T) error {
	if d.size == d.capacity { return ErrFull }
	d.head = (d.head - 1 + d.capacity) % d.capacity
	d.data[d.head] = val
	d.size++
	return nil
}

func (d *Dequeue[T]) PushBack(vals ...T) error {
	for _, val := range vals {
		if d.size == d.capacity { return ErrFull }
		d.data[d.tail] = val
		d.tail = (d.tail + 1) % d.capacity
		d.size++
	}
	return nil
}

func (d *Dequeue[T]) PopFront() (T, error) {
	if d.size == 0 { var zero T; return zero, ErrEmpty }
	oldHead := d.head
	d.head = (d.head + 1) % d.capacity
	d.size--
	return d.data[oldHead], nil
}

func (d *Dequeue[T]) PopBack() (T, error) {
	if d.size == 0 { var zero T; return zero, ErrEmpty }
	oldTail := d.tail
	d.tail = (d.tail - 1 + d.capacity) % d.capacity
	d.size--
	return d.data[oldTail], nil
}

func (d *Dequeue[T]) Front() (T, error) {
	if d.size == 0 { var zero T; return zero, ErrEmpty }
	return d.data[d.head], nil
}

func (d *Dequeue[T]) Back() (T, error) {
	if d.size == 0 { var zero T; return zero, ErrEmpty }
	return d.data[(d.tail - 1 + d.capacity) % d.capacity], nil
}

type MultiRowState struct {
    fgColor, bgColor uint32
    XPos, YPos, maxRows, winWidth, winHeight int
    Instructions *Dequeue[string]
}
func InterpretXTerm(state *MultiRowState, code string) {
	for _, str := range parseXTerm(code) {
		if strings.HasPrefix(str, "\x1b[") {
			switch str[len(str) - 1] {
			case 'H':
				arr := strings.Split(str[2:len(str)-1], ";")
				if len(arr) < 2 { continue }
				state.Instructions.PushBack("XPos="+arr[1], "YPos="+arr[0])
			case 'K': state.Instructions.PushBack("Clear")
			case 'm': state.Instructions.PushBack("XTerm="+str)
			}
		} else if str == "\n" {
			state.Instructions.PushBack("Newline")
		} else {
			state.Instructions.PushBack("<-" + str)
		}
	}
}

func parseXTerm(input string) (ret []string) {
	current, inEscape, inCSI, skipNext := strings.Builder{}, false, false, false // CSI = Control Sequence Introducer
	for i, r := range input {
		if skipNext { skipNext = false; continue }
		switch {
		case !inEscape && r == '\n': // Start of escape sequence
			if current.Len() > 0 {
				ret = append(ret, current.String())
				current.Reset()
			}
			ret = append(ret, "\n")
		case !inEscape && r == 0x1B: // Start of escape sequence
			if current.Len() > 0 {
				ret = append(ret, current.String())
				current.Reset()
			}
			inEscape = true
			current.WriteRune(r)
			if i+1 < len(input) && input[i+1] == '[' {
				inCSI, skipNext = true, true
				current.WriteByte('[')
			}
		case inEscape && !inCSI:
			current.WriteRune(r)
			ret, inEscape = append(ret, current.String()), false
			current.Reset()
		case inCSI:
			current.WriteRune(r)
			switch r {
			case 'm', 'H', 'f', 'A', 'B', 'C', 'D', 's', 'u', 'K', 'J', 'h', 'l':
				ret, inEscape, inCSI = append(ret, current.String()), false, false
				current.Reset()
			}
		default: current.WriteRune(r)
		}
	}
	if current.Len() > 0 { ret = append(ret, current.String()) }
	return
}

func MultiRowGlyphWidget(title string, left, top, winWidth, winHeight int, keypress func(byte, *MultiRowState) int, init func(*MultiRowState)) {
    maxRows := winHeight / GlyphHeight
    state := MultiRowState{
        XPos: 0, YPos: 1, fgColor: 0xffd7afaf, bgColor: 0xff5f5f87,
        Instructions: NewDequeue[string](winWidth/GlyphWidth*maxRows*2),
        maxRows: maxRows, winWidth: winWidth, winHeight: winHeight,
    }
    state.Instructions.PushBack("ClearAll")
	var ximg *XImage
	drawRune := func (aRune uint32) { 
		if state.YPos >= state.maxRows || state.XPos >= winWidth { return }
		glyph := GetColoredGlyph(aRune, state.fgColor, state.bgColor)
		ximg.XDraw(glyph, state.XPos, (state.YPos-1)*GlyphHeight)
		state.XPos += glyph.Width
	}
    interpret := func(instruction string) {
        switch instruction {
		case "Newline": state.XPos = 0; if state.YPos < state.maxRows { state.YPos += 1 }
		case "Backspace": if state.XPos >= GlyphWidth { state.XPos -= GlyphWidth; ximg.XDraw(BlankImage(GlyphWidth, GlyphHeight), state.XPos, (state.YPos-1)*GlyphHeight) }
        case "Clear": if state.XPos<0 || state.XPos>=state.winWidth { return }; ximg.XDraw(BlankImage(state.winWidth-state.XPos, GlyphHeight), state.XPos, (state.YPos-1)*GlyphHeight)
        case "ClearAll": state.XPos, state.YPos = 0, 0; ximg.XDraw(BlankImage(state.winWidth, state.winHeight), 0, 0)
        case "ClearRest": ximg.XDraw(BlankImage(state.winWidth, state.winHeight - state.YPos*GlyphHeight), 0, state.YPos*GlyphHeight)
        default:
            if strings.HasPrefix(instruction, "<-") { ForeachRune([]byte(instruction[2:]), func(aRune uint32){drawRune(aRune)}) }
			if len(instruction) < 5 { return }
			switch instruction[:5] {
			case "YPos=": if y := ParseInt(instruction[5:]); y >= 1 && y <= state.maxRows { state.YPos = y }
			case "XPos=": if x := ParseInt(instruction[5:]); x >= 0 && x * GlyphWidth <= state.winWidth { state.XPos = x * GlyphWidth }
			case "XTerm": parseXTermColor(&state, strings.TrimPrefix(instruction, "XTerm="))
			}
        }
    }
    UniversalWidget(title, left, top, winWidth, winHeight, func(ximg *XImage) (int, int) {
        for {
            if state.Instructions.size == 0 { break }
            instruction, err := state.Instructions.PopFront()
            if err == nil { interpret(instruction) }
        }
        return 0, 0
    }, nil, func(detail byte) int {
        if keypress == nil { return -1 }
        return keypress(detail, &state)
    }, nil, func(xim *XImage) {
		WindowRaiseFocuser(xim)
		ximg = xim
		if init != nil { init(&state) }
	})
}


type SingleRowState struct { XPos int; Instructions *Dequeue[string] }
func SingleRowGlyphWidget(title string, left, top, winWidth int, modKeys []uint16, keypress func (byte, *SingleRowState) int, init func(*SingleRowState)) {
	state := SingleRowState { XPos: 0, Instructions: NewDequeue[string](winWidth / GlyphWidth * 2) }
	state.Instructions.PushBack("Clear")
	var ximg *XImage
	var XPosBackup int
	interpret := func(instruction string) {
		switch instruction {
		case "Clear": state.XPos = 0; ximg.XDraw(BlankImage(winWidth, GlyphHeight), 0, 0)
		case "XPos#Save": XPosBackup = state.XPos
		case "XPos#Load": state.XPos = XPosBackup 
		case "Ungrab#Backspace": ximg.Ungrab(22)
		case "Backspace": if state.XPos >= GlyphWidth  { state.XPos -= GlyphWidth; ximg.XDraw(BlankImage(GlyphWidth, GlyphHeight), state.XPos, 0) }
		case "Raise": RaiseWindow(ximg.Win)
		case "SetIM": ImWindow = ximg.Win
		case "Grab#Backspace": ximg.Grab(0, 22)
		case "Grab#Return": ximg.Grab(0, 36)
		case "Grab#IM": for _, code := range []byte{9, 10, 11, 12, 13, 14, 20, 21, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 38, 39, 40, 41, 42, 43, 44, 45, 46, 52, 53, 54, 55, 56, 57, 58, 88, 89, 90, 91, 92} { for _, mod := range modKeys { ximg.Grab(mod, code) } }
		default: glyph := GetColoredGlyph(StringToRune(instruction), 0xffd7afaf, 0xff5f5f87); if state.XPos + glyph.Width < winWidth { ximg.XDraw(glyph, state.XPos, 0); state.XPos += glyph.Width }
		}
	}
	UniversalWidget(title, left, top, winWidth, GlyphHeight, func (ximg *XImage) (int, int) {
		for {
			if state.Instructions.size == 0 { break }
			instruction, err := state.Instructions.PopFront()
			if err == nil { interpret(instruction) }
		}
		return 0, 0
	}, nil, func(detail byte) int {
		if keypress == nil { return 0 }
		return keypress(detail, &state)
	}, nil, func (xim *XImage) { 
		ximg = xim
		if init != nil { init(&state) }
	})
}

func SimpleCanvasWidget(title string, img RGBAData) {
    top, left := 0, 0
    UniversalWidget(title, 0, 0, Width, Height, func(ximg *XImage) (int, int) {
        if img.Width - left < Width { left = img.Width - Width }
        if left < 0 { left = 0 }
        if img.Height - top < Height { top = img.Height - Height }
        if top < 0 { top = 0 } 
        w, h := Width, Height
        if img.Width - left < w { w = img.Width - left }
        if img.Height - top < h { h = img.Height - top }
        ximg.XDraw(Crop(img, left, top, w, h), 0, 0)
        return 0, 0
    }, func (detail byte, x, y int16) int {
        switch detail {
        case 4: top -= 200
        case 5: top += 200
        case 6: left -= 200
        case 7: left += 200
        default: return 0
        }
        return 1
    }, func (detail byte) int {
        switch detail {
        case 24: return -1 //"q"
        case 111: top -= 200
        case 116: top += 200
        case 113: left -=200
        case 114: left +=200
        default: return 0
        }
        return 1
    }, nil, WindowRaiseFocuser)
}
