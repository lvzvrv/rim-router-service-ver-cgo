package utils

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	bracketLineRe = regexp.MustCompile(`^\[(?P<ts>[^]]+)\]\s*\[(?P<level>[^]]+)\]\s*(?P<msg>.*)$`)
	timeAltLayout = "2006-01-02 15:04:05,000" // формат [2025-09-19 04:37:38,155]
)

// ReadTailLines читает последние n строк файла эффективно (с конца).
func ReadTailLines(f *os.File, n int) ([]string, error) {
	const chunk = 4 * 1024
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return []string{}, nil
	}

	var buf []byte
	var read int64
	for read < size && strings.Count(string(buf), "\n") <= n {
		step := int64(chunk)
		if read+step > size {
			step = size - read
		}
		read += step
		tmp := make([]byte, step)
		_, err := f.ReadAt(tmp, size-read)
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		buf = append(tmp, buf...)
	}
	lines := strings.Split(string(buf), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// ParseBracketLine преобразует строку вида
// [2025-09-19 04:37:38,155] [ERROR] ModbusServiceFunctions::ReadInt: Read timeout
// в JSON-совместимый map.
func ParseBracketLine(s string) map[string]any {
	m := bracketLineRe.FindStringSubmatch(s)
	if len(m) == 0 {
		return map[string]any{"raw": s}
	}

	fields := map[string]string{}
	for i, name := range bracketLineRe.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		fields[name] = strings.TrimSpace(m[i])
	}

	ts := fields["ts"]
	if t, err := time.Parse(timeAltLayout, ts); err == nil {
		ts = t.UTC().Format(time.RFC3339Nano)
	}

	msg := fields["msg"]
	module := ""
	if idx := strings.Index(msg, "::"); idx > 0 {
		modpart := msg[:idx]
		rest := msg[idx+2:]
		if cidx := strings.Index(rest, ":"); cidx >= 0 && cidx < 40 {
			msg = strings.TrimSpace(rest[cidx+1:])
			module = strings.TrimSpace(modpart)
		}
	}

	return map[string]any{
		"time":    ts,
		"level":   strings.ToLower(fields["level"]),
		"module":  module,
		"message": msg,
		"raw":     s,
	}
}

// StreamLines читает строки из io.Reader построчно в канал.
func StreamLines(r io.Reader, ch chan<- string) {
	defer close(ch)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		ch <- sc.Text()
	}
}
