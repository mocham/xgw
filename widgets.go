package xgw
import "strings"
func universalWidget(title string, left, top, winWidth, winHeight int, paint func (*XImage) (int, int), button func (byte, int16, int16) int, keypress func (byte) int, refresh func(string), init func(*XImage)) {
	ximg := newXImage(left, top, winWidth, winHeight, title)
	if ximg == nil { return }
	defer func() { ximg.Ungrab(0); ximg.Destroy() }()
	if init != nil { init(ximg) }
	paintWrap := func () {
		w, h := paint(ximg)
		if w >0 && h > 0 { resizeWindow(ximg.Win, left, top, w, h) }
		ximg.Flush()
	}
	paintWrap()
	for {
        ev, err := ximg.Conn.WaitForEvent()
        if err != nil || ev == nil { continue }
        switch event := ev.(type) {
		case EXProp:
			if refresh == nil { continue }
			if newTitle := getTitle(ximg.Win); len(newTitle)>0 && newTitle[len(newTitle)-1] == '*' {
				refresh(newTitle)
				paintWrap()
				setWmName(ximg.Win, title)
			}
        case EXClient: if event.Type==atomMap["WM_PROTOCOLS"] && event.Data.Data32[0]==uint32(atomMap["WM_DELETE_WINDOW"]) && ximg.Win==event.Window { return }
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
			if int(detail) > len(xconf.Keymap) { detail = 0 }
			switch keypress(detail) {
			case 1: paintWrap()
			case -1: return
			}
		}
	}
}
func windowRaiseFocuser(ximg *XImage) { raiseWindow(ximg.Win); focusSet(ximg.Win) }

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

type multiRowState struct {
    fgColor, bgColor uint32
    xPos, yPos, maxRows, winWidth, winHeight int
    instructions *Dequeue[string]
}
func interpretXTerm(state *multiRowState, code string) {
	for _, str := range parseXTerm(code) {
		if strings.HasPrefix(str, "\x1b[") {
			switch str[len(str) - 1] {
			case 'H':
				arr := strings.Split(str[2:len(str)-1], ";")
				if len(arr) < 2 { continue }
				state.instructions.PushBack("xPos="+arr[1], "yPos="+arr[0])
			case 'K': state.instructions.PushBack("Clear")
			case 'm': state.instructions.PushBack("XTerm="+str)
			}
		} else if str == "\n" {
			state.instructions.PushBack("Newline")
		} else {
			state.instructions.PushBack("<-" + str)
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

func multiRowGlyphWidget(title string, left, top, winWidth, winHeight int, keypress func(byte, *multiRowState) int, init func(*multiRowState)) {
    maxRows := winHeight / glyphHeight
    state := multiRowState{
        xPos: 0, yPos: 1, fgColor: 0xffd7afaf, bgColor: 0xff5f5f87,
        instructions: NewDequeue[string](winWidth/glyphWidth*maxRows*2),
        maxRows: maxRows, winWidth: winWidth, winHeight: winHeight,
    }
    state.instructions.PushBack("ClearAll")
	var ximg *XImage
	drawRune := func (aRune uint32) { 
		if state.yPos >= state.maxRows || state.xPos >= winWidth { return }
		glyph := getColoredGlyph(aRune, state.fgColor, state.bgColor)
		ximg.XDraw(glyph, state.xPos, (state.yPos-1)*glyphHeight)
		state.xPos += glyph.Width
	}
    interpret := func(instruction string) {
        switch instruction {
		case "Newline": state.xPos = 0; if state.yPos < state.maxRows { state.yPos += 1 }
		case "Backspace": if state.xPos >= glyphWidth { state.xPos -= glyphWidth; ximg.XDraw(blankImage(glyphWidth, glyphHeight), state.xPos, (state.yPos-1)*glyphHeight) }
        case "Clear": if state.xPos<0 || state.xPos>=state.winWidth { return }; ximg.XDraw(blankImage(state.winWidth-state.xPos, glyphHeight), state.xPos, (state.yPos-1)*glyphHeight)
        case "ClearAll": state.xPos, state.yPos = 0, 0; ximg.XDraw(blankImage(state.winWidth, state.winHeight), 0, 0)
        case "ClearRest": ximg.XDraw(blankImage(state.winWidth, state.winHeight - state.yPos*glyphHeight), 0, state.yPos*glyphHeight)
        default:
            if strings.HasPrefix(instruction, "<-") { foreachRune([]byte(instruction[2:]), func(aRune uint32){drawRune(aRune)}) }
			if len(instruction) < 5 { return }
			switch instruction[:5] {
			case "yPos=": if y := parseInt(instruction[5:]); y >= 1 && y <= state.maxRows { state.yPos = y }
			case "xPos=": if x := parseInt(instruction[5:]); x >= 0 && x * glyphWidth <= state.winWidth { state.xPos = x * glyphWidth }
			case "XTerm": parseXTermColor(&state, strings.TrimPrefix(instruction, "XTerm="))
			}
        }
    }
    universalWidget(title, left, top, winWidth, winHeight, func(ximg *XImage) (int, int) {
        for {
            if state.instructions.size == 0 { break }
            instruction, err := state.instructions.PopFront()
            if err == nil { interpret(instruction) }
        }
        return 0, 0
    }, nil, func(detail byte) int {
        if keypress == nil { return -1 }
        return keypress(detail, &state)
    }, nil, func(xim *XImage) {
		windowRaiseFocuser(xim)
		ximg = xim
		if init != nil { init(&state) }
	})
}
