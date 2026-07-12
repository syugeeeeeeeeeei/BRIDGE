package yamlmini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type line struct {
	indent int
	text   string
	number int
}

func ToJSON(data []byte) ([]byte, error) {
	var lines []line
	s := bufio.NewScanner(strings.NewReader(string(data)))
	for n := 1; s.Scan(); n++ {
		raw := strings.TrimRight(s.Text(), " \t\r")
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}
		indent := 0
		for indent < len(raw) && raw[indent] == ' ' {
			indent++
		}
		if indent%2 != 0 {
			return nil, fmt.Errorf("yaml line %d: indentation must use multiples of two spaces", n)
		}
		text := stripComment(strings.TrimSpace(raw))
		lines = append(lines, line{indent: indent, text: text, number: n})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty YAML")
	}
	v, next, err := parseBlock(lines, 0, lines[0].indent)
	if err != nil {
		return nil, err
	}
	if next != len(lines) {
		return nil, fmt.Errorf("yaml line %d: unexpected content", lines[next].number)
	}
	return json.Marshal(v)
}

func parseBlock(lines []line, i, indent int) (any, int, error) {
	if i >= len(lines) || lines[i].indent != indent {
		return nil, i, fmt.Errorf("yaml line %d: invalid indentation", lines[i].number)
	}
	if strings.HasPrefix(lines[i].text, "- ") || lines[i].text == "-" {
		return parseSeq(lines, i, indent)
	}
	return parseMap(lines, i, indent)
}
func parseMap(lines []line, i, indent int) (any, int, error) {
	m := map[string]any{}
	for i < len(lines) && lines[i].indent == indent && !strings.HasPrefix(lines[i].text, "-") {
		key, rest, ok := strings.Cut(lines[i].text, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, i, fmt.Errorf("yaml line %d: expected key: value", lines[i].number)
		}
		key = strings.TrimSpace(key)
		if _, exists := m[key]; exists {
			return nil, i, fmt.Errorf("yaml line %d: duplicate key %q", lines[i].number, key)
		}
		rest = strings.TrimSpace(rest)
		i++
		if rest != "" {
			v, err := scalar(rest)
			if err != nil {
				return nil, i, fmt.Errorf("yaml: %w", err)
			}
			m[key] = v
			continue
		}
		if i >= len(lines) || lines[i].indent <= indent {
			m[key] = map[string]any{}
			continue
		}
		v, next, err := parseBlock(lines, i, lines[i].indent)
		if err != nil {
			return nil, i, err
		}
		m[key] = v
		i = next
	}
	return m, i, nil
}
func parseSeq(lines []line, i, indent int) (any, int, error) {
	arr := []any{}
	for i < len(lines) && lines[i].indent == indent && (strings.HasPrefix(lines[i].text, "- ") || lines[i].text == "-") {
		text := strings.TrimSpace(strings.TrimPrefix(lines[i].text, "-"))
		lineNo := lines[i].number
		i++
		if text == "" {
			if i >= len(lines) || lines[i].indent <= indent {
				return nil, i, fmt.Errorf("yaml line %d: empty list item", lineNo)
			}
			v, next, err := parseBlock(lines, i, lines[i].indent)
			if err != nil {
				return nil, i, err
			}
			arr = append(arr, v)
			i = next
			continue
		}
		if key, rest, ok := strings.Cut(text, ":"); ok {
			m := map[string]any{}
			key = strings.TrimSpace(key)
			rest = strings.TrimSpace(rest)
			if rest != "" {
				v, err := scalar(rest)
				if err != nil {
					return nil, i, err
				}
				m[key] = v
			} else if i < len(lines) && lines[i].indent > indent {
				v, next, err := parseBlock(lines, i, lines[i].indent)
				if err != nil {
					return nil, i, err
				}
				m[key] = v
				i = next
			} else {
				m[key] = map[string]any{}
			}
			for i < len(lines) && lines[i].indent > indent {
				if lines[i].indent != indent+2 || strings.HasPrefix(lines[i].text, "-") {
					break
				}
				k, r, ok := strings.Cut(lines[i].text, ":")
				if !ok {
					return nil, i, fmt.Errorf("yaml line %d: expected key: value", lines[i].number)
				}
				k = strings.TrimSpace(k)
				r = strings.TrimSpace(r)
				if _, exists := m[k]; exists {
					return nil, i, fmt.Errorf("yaml line %d: duplicate key %q", lines[i].number, k)
				}
				i++
				if r != "" {
					v, err := scalar(r)
					if err != nil {
						return nil, i, err
					}
					m[k] = v
				} else if i < len(lines) && lines[i].indent > indent+2 {
					v, next, err := parseBlock(lines, i, lines[i].indent)
					if err != nil {
						return nil, i, err
					}
					m[k] = v
					i = next
				} else {
					m[k] = map[string]any{}
				}
			}
			arr = append(arr, m)
			continue
		}
		v, err := scalar(text)
		if err != nil {
			return nil, i, err
		}
		arr = append(arr, v)
	}
	return arr, i, nil
}
func scalar(s string) (any, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []any{}, nil
		}
		parts := strings.Split(inner, ",")
		a := make([]any, 0, len(parts))
		for _, p := range parts {
			v, err := scalar(p)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		return a, nil
	}
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) || (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		return s[1 : len(s)-1], nil
	}
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "~":
		return nil, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return s, nil
}
func stripComment(s string) string {
	inSingle, inDouble := false, false
	for i, r := range s {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || s[i-1] == ' ') {
				return strings.TrimSpace(s[:i])
			}
		}
	}
	return s
}
