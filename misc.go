package xgw
import (
	"fmt"
    "math"
	"unicode"
	"os"
	"strings"
	"strconv"
	"path/filepath"
)
func parseInt(s string) (ret int) { ret, _ = strconv.Atoi(s); return }
func fmtInt(i int) string { return fmt.Sprintf("%d", i) }
func fmtChar(c rune) string { return fmt.Sprintf("%c", c) }
func HasPreChar(s string, b byte) bool { return len(s) >= 1 && s[0] == b }
func CStrBytes(str string) (ret []byte) { ret = make([]byte, len(str)+1); copy(ret, str); return }
func removeElement[T comparable](arr []T, ele T) []T { for i, v := range arr { if v == ele { return append(arr[:i], arr[i+1:]...) } }; return arr }
func Abs[T ~int](x T) T { if x < 0 { return -x }; return x }
func Abs32[T ~int](x T) uint32 { if x < 0 { return uint32(-x) }; return uint32(x) }
func isPrintable(b byte) bool { return b >= 32 && b <= 126 }
func stringToRune(s string) uint32 { var buf [4]byte; copy(buf[:], s); return *Ptr[uint32](&buf[0]) }
func HexToUint32(hex string) uint32 { rgb, _ := strconv.ParseUint(hex, 16, 32); return uint32(0xFF000000 | rgb) }
func isDir(path string) bool { fileInfo, err := os.Stat(path); return err == nil && fileInfo.IsDir() }
func DbEscape(path string) string { return strings.ReplaceAll(strings.ReplaceAll(path, "_", `\_`), "%", `\%`) }
func replaceAlls(s []string, replacements map[string]string) (ret[]string) { for _, st := range s { ret = append(ret, replaceAll(st, replacements)) }; return }

func foreachRune(b []byte, callback func (uint32)) {
	for i := 0; i < len(b); {
		switch b[i] >> 4 {
		case 12, 13: callback(uint32(b[i])|uint32(b[i+1])<<8); i += 2
		case 14: callback(uint32(b[i])|uint32(b[i+1])<<8|uint32(b[i+2])<<16); i += 3
		case 15: callback(uint32(b[i])|uint32(b[i+1])<<8|uint32(b[i+2])<<16|uint32(b[i+3])<<24); i += 4
		default: callback(uint32(b[i])); i++
		}
	}
}

func replaceAll(s string, replacements map[string]string) string {
    b, parts := strings.Builder{}, strings.Split(s, "${")
    b.WriteString(parts[0])
    for _, part := range parts[1:] {
        end := strings.Index(part, "}")
		if end < 0 { continue }
		b.WriteString(replacements[part[:end]])
		b.WriteString(part[end+1:])
    }
    return b.String()
}

func findMissingInteger(names []string) int {
    nameSet := make(map[string]struct{}, len(names))
    for _, name := range names { nameSet[name] = struct{}{} }
    for k := 1; ; k++ { if _, exists := nameSet[fmtInt(k)]; !exists { return k } }
}

func extractPrintable(input string) string {
	var result []rune
	for _, r := range input { if unicode.IsPrint(r) || r == '\n' { result = append(result, r) } }
	return string(result)
}

func humanize(size float64) string {
    units, threshold, unitIndex := "KMGTP", 1024.0, 0
    for size >= threshold && unitIndex < len(units)-1 { size /= threshold; unitIndex++ }
    switch {
    case size >= 1000: size /= threshold; return fmt.Sprintf("%.2f%c", size, units[unitIndex])
    case size >= 100: return fmt.Sprintf(" %d%c", int(math.Round(size)), units[unitIndex])
    case size >= 10: return fmt.Sprintf("%.1f%c", size, units[unitIndex])
    default: return fmt.Sprintf("%.2f%c", size, units[unitIndex])
    }
}

func congruentMod(n, size int) int {
    if size <= 0 { return 0 }
    if m := n % size; m < 0 { return m+size } else { return m }
}

func expandHome(relativePath string) string {
	if strings.HasPrefix(relativePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil { return relativePath }
		return filepath.Join(homeDir, relativePath[1:])
	}
	return relativePath
}

func createOrReplaceSymlink(target, link string) error {
    if _, err := os.Lstat(link); err == nil {
        if err := os.Remove(link); err != nil { return err }
    } else { return err }
    return os.Symlink(target, link)
}

func findFilesWithSubstring(dir, substr string) (matches []string) {
    filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err == nil && !info.IsDir() && strings.Contains(info.Name(), substr) { matches = append(matches, path) }
        return err
    })
    return
}

func matchOSFiles(pattern string) (matches []string, err error) {
	if strings.HasPrefix(pattern, "~/") {
		home, err := os.UserHomeDir()
		if err != nil { return nil, err }
		pattern = filepath.Join(home, pattern[2:])
	} 
	dir, filePattern := filepath.Split(pattern)
	if dir == "" { dir, _ = filepath.Abs(".") }
	files, err := os.ReadDir(dir)
	if err != nil { return }
	for _, file := range files {
		matched, err := filepath.Match(filePattern, file.Name())
		if filePattern == file.Name() { err = nil; matched = true }
		if err != nil { return nil, err }
		if matched { matches = append(matches, filepath.Join(dir, file.Name())) }
	}
	if len(matches) == 0 { err = ErrNotFound }
	return
}
