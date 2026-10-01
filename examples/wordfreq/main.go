// goalchemy:unordered

// Command wordfreq counts words in a text and prints the most frequent ones.
package main

const text = `It was the best of times, it was the worst of times, it was the age of
wisdom, it was the age of foolishness, it was the epoch of belief, it was the
epoch of incredulity, it was the season of Light, it was the season of Darkness`

type entry struct {
	word  string
	count int
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

func words(s string) []string {
	var out []string
	start := -1
	for i := 0; i <= len(s); i++ {
		if i < len(s) && isLetter(s[i]) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, lower(s[start:i]))
			start = -1
		}
	}
	return out
}

func less(a, b entry) bool {
	if a.count != b.count {
		return a.count > b.count
	}
	return a.word < b.word
}

func sortEntries(es []entry) {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && less(es[j], es[j-1]); j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
}

func main() {
	counts := map[string]int{}
	for _, w := range words(text) {
		counts[w]++
	}
	var es []entry
	for w, c := range counts {
		es = append(es, entry{w, c})
	}
	sortEntries(es)
	for i, e := range es {
		if i == 5 {
			break
		}
		println(e.word, e.count)
	}
	println("distinct words:", len(es))
}
