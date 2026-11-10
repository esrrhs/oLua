package main

import "strings"

// sourceToken preserves byte offsets, excluding comments and whitespace. String
// literals (including long brackets) are single opaque tokens. This is a source
// editing aid; the Lua parser remains the authority on syntax.
type sourceToken struct {
	text             string
	start, end, line int
	literal          bool
}

func luaSourceTokens(s string) []sourceToken {
	var out []sourceToken
	line := 1
	longEnd := func(at int) int {
		if at >= len(s) || s[at] != '[' {
			return -1
		}
		j := at + 1
		for j < len(s) && s[j] == '=' {
			j++
		}
		if j >= len(s) || s[j] != '[' {
			return -1
		}
		close := "]" + s[at+1:j] + "]"
		n := strings.Index(s[j+1:], close)
		if n < 0 {
			return len(s)
		}
		return j + 1 + n + len(close)
	}
	for i := 0; i < len(s); {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			if c == '\n' {
				line++
			}
			i++
			continue
		}
		if strings.HasPrefix(s[i:], "--") {
			end := longEnd(i + 2)
			if end < 0 {
				end = i + 2
				for end < len(s) && s[end] != '\n' {
					end++
				}
			}
			line += strings.Count(s[i:end], "\n")
			i = end
			continue
		}
		start, ln := i, line
		literal := false
		if c == '\'' || c == '"' {
			literal = true
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i++
					if i < len(s) {
						i++
					}
					continue
				}
				if s[i] == c {
					i++
					break
				}
				i++
			}
		} else if end := longEnd(i); end >= 0 {
			literal = true
			i = end
		} else if isLuaIdentStart(c) {
			i++
			for i < len(s) && isLuaIdentPart(s[i]) {
				i++
			}
		} else if c >= '0' && c <= '9' || c == '.' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			i++
			for i < len(s) {
				d := s[i]
				if isLuaIdentPart(d) || d == '.' {
					i++
					continue
				}
				if (d == '+' || d == '-') && (s[i-1] == 'e' || s[i-1] == 'E' || s[i-1] == 'p' || s[i-1] == 'P') {
					i++
					continue
				}
				break
			}
		} else {
			i++
			for _, op := range []string{"...", "..", "==", "~=", "<=", ">=", "//", "<<", ">>", "::"} {
				if strings.HasPrefix(s[start:], op) {
					i = start + len(op)
					break
				}
			}
		}
		line += strings.Count(s[start:i], "\n")
		out = append(out, sourceToken{s[start:i], start, i, ln, literal})
	}
	return out
}
func isLuaIdentStart(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isLuaIdentPart(c byte) bool  { return isLuaIdentStart(c) || c >= '0' && c <= '9' }

// replaceCodePath never rewrites comments, strings, method names, or suffixes of
// other paths. Complete token sequences tolerate whitespace around dots.
func replaceCodePath(source, path, replacement string) string {
	ts, ps := luaSourceTokens(source), luaSourceTokens(path)
	for i := len(ts) - len(ps); i >= 0; i-- {
		if len(ps) == 0 {
			break
		}
		if len(ps) == 1 && i+1 < len(ts) && ts[i+1].text == "=" {
			continue
		} // named constructor key / assignment
		if i > 0 && (ts[i-1].text == "." || ts[i-1].text == ":" || ts[i-1].text == "::" || ts[i-1].text == "goto") {
			continue
		}
		matches := true
		for j, p := range ps {
			if ts[i+j].literal || ts[i+j].text != p.text {
				matches = false
				break
			}
		}
		if matches {
			source = source[:ts[i].start] + replacement + source[ts[i+len(ps)-1].end:]
		}
	}
	return source
}

func isLuaKeyword(s string) bool {
	switch s {
	case "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while":
		return true
	}
	return false
}
