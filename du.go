package xgw
import (
	"path/filepath"
	"strings"
)
func xterm256ToARGB(code int) uint32 {
    if code < 0 || code > 255 { return 0xFF000000 }
    if code <= 15 { return uint32(xconf.XTermColors[code]) }
    if code >= 232 { gray := uint32(8 + 10*(code-232)); return 0xFF000000 | (gray << 16) | (gray << 8) | gray} // Handle grayscale (232-255)
	code -= 16 // Handle 6x6x6 color cube (16-231)
	scale := func(v int) uint32 { if v > 0 { return uint32(55 + 40*v) }; return 0}
	return 0xFF000000 | (scale(code / 36) << 16) | (scale((code % 36) / 6) << 8) | scale(code % 6)
}
func parseXTermColor(state *multiRowState, escapeSeq string) {
	if len(escapeSeq) < 4 { return }
	if escapeSeq == "\x1b[0m" { state.bgColor, state.fgColor = 0, 0xff8787af; return }
	if parts := strings.Split(escapeSeq[2:len(escapeSeq)-1], ";"); len(parts) == 1 {
		if code := ParseInt(parts[0]); code >= 30 && code <= 37 {
			state.fgColor = xterm256ToARGB(code - 30)
		} else if code >= 40 && code <= 47 {
			state.bgColor = xterm256ToARGB(code - 40)
		}
	} else if len(parts) == 3 && parts[1] == "5" {
        if code := ParseInt(parts[2]); parts[0] == "48" {
			state.bgColor = xterm256ToARGB(code)
		} else {
			state.fgColor = xterm256ToARGB(code)
		}
    }
}
type TDu interface { int64 | int | string } 
type Pair[T TDu] struct {Key string; Value T}
type DuState[T TDu] struct {
	Path string; Cursor, Left, Right, Rows int
	List []Pair[T]; Dict map[string]T
	Render func(string, map[string]T, bool) string
	At func(string, string) string
}

func duRenderWithOffset[T TDu](state *DuState[T], cursor int) (ret string) {
    if cursor < 0 || cursor >= len(state.List) || cursor < state.Left || cursor >= state.Right { return }
    pref := FmtInt(cursor)
    if cursor < 10 { pref = " " + pref }
	ret = "\x1b[" + FmtInt(cursor-state.Left+1) + ";0H" + pref + "│" + state.Render(filepath.Join(state.Path, state.List[cursor].Key), state.Dict, cursor==0) + "\x1b[K\x1b[0m"
	if cursor == 0 { ret = "\x1b[48;5;60m" + ret }
	return
}

func duRefresh[T TDu](state *DuState[T]) (ret string) {
	for i := state.Left; i < state.Right; i++ {
		if i == state.Cursor { ret += "\x1b[38;5;208m" }
		ret += duRenderWithOffset[T](state, i)
	}
	return
}

func duUpdate[T TDu](state *DuState[T], oldCursor int) string {
	state.Cursor = CongruentMod(state.Cursor, len(state.List)) 
	height := state.Right - state.Left
	switch {
	case state.Cursor < state.Left: state.Left, state.Right = state.Cursor, state.Cursor+height; return duRefresh[T](state)
	case state.Cursor >= state.Right: state.Left, state.Right = state.Cursor+1-height, state.Cursor+1; return duRefresh[T](state)
	default: return duRenderWithOffset[T](state, oldCursor) + "\x1b[38;5;208m" + duRenderWithOffset[T](state, state.Cursor)
	}
}

func duAt[T TDu](state *DuState[T]) string {
	if state.Cursor < 0 || state.Cursor >= len(state.List) { return "" }
	return state.At(state.Path, state.List[state.Cursor].Key)
}

func DuWidget[T TDu](path, sortName string, widthPerc float64, Query func(string, string) *DuState[T], Run func (*DuState[T], string) string) {
	var duState *DuState[T]
    winWidth, cmd, cmdPos, cursors := int(float64(width) * widthPerc), "", "xPos=" + FmtInt(15*scale), make(map[string]int)
	if winWidth < 0 { winWidth = 300 }
    init := func (state *multiRowState) {
		if duState != nil { cursors[duState.Path] = duState.Cursor }
		newState := Query(path, sortName)
		if newState == nil { path = duState.Path; return }
		duState = newState
		duState.Right = state.maxRows
        if cursorBackup, exists := cursors[path]; exists { duState.Cursor = cursorBackup }
        interpretXTerm(state, duRefresh(duState))
		state.instructions.PushBack("ClearRest")
    }
    multiRowGlyphWidget("auto-du-widget", width - winWidth, 0, winWidth, height - 60, func (detail byte, state *multiRowState) (ret int) {
        oldCursor := duState.Cursor
		ret = 1
        if cmd != "" {
			switch detail{
			case 9: cmd = ""; state.instructions.PushBack(cmdPos, "Clear")
			case 22: 
                state.instructions.PushBack("Backspace")
                cmd = cmd[:len(cmd)-1]
                if cmd == "" { state.instructions.PushBack(cmdPos, "Clear") }
			case 36:
                newPath := Run(duState, cmd)
                cmd = ""
				if newPath == "" { return 0 }
				path = newPath; init(state)
            default:
                ch := xconf.Keymap[detail]
				if len(ch) > 4 || ch == "N/A" { return 0 }
                cmd += ch
                state.instructions.PushBack("<-" + ch)
            }
            return
        }
        switch detail {
        case 9, 24: return -1 // ESC, Q
        case 114:
            newPath := duAt(duState)
            if newPath == "" { return 0 }
            path = newPath; init(state)
        case 113:
            newPath := duState.At(duState.Path, "..")
            if newPath == "" { return 0 }
            path = newPath; init(state)
        case 57: sortName = "name"; init(state)
        case 39: sortName = "size"; init(state)
        case 61: cmd = "/"; state.instructions.PushBack(cmdPos, "<-/")
        case 40, 54, 111, 116:
			delta := (int(detail)-113)/2
			if Abs(delta) > 2 { delta = -2*(int(detail)-47) }
            duState.Cursor += delta
            interpretXTerm(state, duUpdate(duState, oldCursor))
        default: 
			ch := xconf.Keymap[int(detail)]
			if ch == "N/A" || len(ch) > 4 { return 0 }
            if cmd == "" { state.instructions.PushBack(cmdPos, "<-:") }
            cmd += ch
            state.instructions.PushBack("<-" + ch)
        }
        return
    }, init)
}
