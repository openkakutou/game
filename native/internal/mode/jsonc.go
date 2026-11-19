package mode

// stripJSONC turns JSONC into JSON: it blanks // and /* */ comments and
// drops trailing commas before ] and }, leaving strings untouched. Blanked
// comments keep their newlines so JSON error offsets stay on the same line.
func stripJSONC(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inString := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(src) {
				i++
				out = append(out, src[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				out = append(out, '\n')
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i < len(src) && !(src[i] == '*' && i+1 < len(src) && src[i+1] == '/') {
				if src[i] == '\n' {
					out = append(out, '\n')
				}
				i++
			}
			i++ // past the closing '/'
			continue
		}
		if c == ']' || c == '}' {
			dropTrailingComma(&out)
		}
		out = append(out, c)
	}
	return out
}

// dropTrailingComma removes a comma that only whitespace separates from the
// end of out.
func dropTrailingComma(out *[]byte) {
	b := *out
	j := len(b) - 1
	for j >= 0 && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n' || b[j] == '\r') {
		j--
	}
	if j >= 0 && b[j] == ',' {
		*out = append(b[:j], b[j+1:]...)
	}
}
