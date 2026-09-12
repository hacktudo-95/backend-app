package rag

import "strings"

// ChunkText groups paragraphs up to maxRunes and carries a small tail overlap.
func ChunkText(text string, maxRunes, overlap int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current []rune
	flush := func() {
		clean := strings.TrimSpace(string(current))
		if clean == "" { return }
		chunks = append(chunks, clean)
		if overlap > len(current) { overlap = len(current) }
		current = append([]rune(nil), current[len(current)-overlap:]...)
	}
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" { continue }
		r := []rune(p)
		for len(r) > 0 {
			space := maxRunes - len(current)
			if space <= 0 { flush(); space = maxRunes - len(current) }
			take := len(r); if take > space { take = space }
			if len(current) > 0 { current = append(current, '\n', '\n') }
			current = append(current, r[:take]...); r = r[take:]
			if len(current) >= maxRunes { flush() }
		}
	}
	if strings.TrimSpace(string(current)) != "" { chunks = append(chunks, strings.TrimSpace(string(current))) }
	return chunks
}
