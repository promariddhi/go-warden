package logging

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

func LogEvent(level, event string, fields map[string]any) {
	var b strings.Builder

	b.WriteString(level)
	b.WriteString(" event=")
	b.WriteString(event)

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(formatValue(fields[k]))
	}

	log.Println(b.String())
}

func formatValue(v any) string {
	switch t := v.(type) {
	case string:
		return quoteIfNeeded(t)
	case int, int64, uint64, bool:
		return fmt.Sprint(t)
	default:
		return quoteIfNeeded(fmt.Sprint(t))
	}
}

func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t\n\"=") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
