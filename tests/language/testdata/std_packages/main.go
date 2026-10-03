package main

import (
	"github.com/eugenioenko/goalchemy/std/bytes"
	"github.com/eugenioenko/goalchemy/std/encoding/binary"
	"github.com/eugenioenko/goalchemy/std/encoding/hex"
	"github.com/eugenioenko/goalchemy/std/sort"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

type word struct {
	text  string
	count int
}

type byCount []word

func (w byCount) Len() int           { return len(w) }
func (w byCount) Less(i, j int) bool { return w[i].count > w[j].count }
func (w byCount) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }

func main() {
	text := "  The quick brown fox, the LAZY dog; the Fox! Ünïcödé ſtraße  "
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !(r >= 'a' && r <= 'z' || r > 127) })
	counts := map[string]int{}
	var order []string
	for _, f := range fields {
		if counts[f] == 0 {
			order = append(order, f)
		}
		counts[f]++
	}
	var ws []word
	for _, f := range order {
		ws = append(ws, word{f, counts[f]})
	}
	sort.Stable(byCount(ws))
	var b strings.Builder
	for i, w := range ws {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(w.text + "=" + strconv.Itoa(w.count))
	}
	println(b.String())
	println(strings.Join(strings.Split("a,b,,c", ","), "|"), strings.Repeat("ab", 3), strings.ReplaceAll("aXbXc", "X", "--"))
	println(strings.TrimSpace(text), strings.EqualFold("Straße", "STRASSE"), strings.EqualFold("ſ", "S"))
	r := strings.NewReplacer("<", "&lt;", ">", "&gt;")
	println(r.Replace("<a href>"), strings.ToUpper("ﬀ ǆ σς"), strings.Count("cheese", "e"), strings.Index("chicken", "ken"))
	before, after, found := strings.Cut("key=value=x", "=")
	println(before, after, found, strings.HasPrefix("golang", "go"), utf8.RuneCountInString("héllo"), utf8.ValidString("\xff"))
	ints := []int{5, -2, 9, 0, 3}
	sort.Ints(ints)
	strs := []string{"pear", "apple", "fig"}
	sort.Strings(strs)
	println(ints[0], ints[4], strs[0], strs[2], sort.SearchInts(ints, 3))
	buf := binary.BigEndian.AppendUint32(nil, 0xDEADBEEF)
	buf = binary.LittleEndian.AppendUint16(buf, 0x1234)
	buf = binary.AppendUvarint(buf, 300)
	println(hex.EncodeToString(buf), binary.BigEndian.Uint32(buf), binary.LittleEndian.Uint16(buf[4:]))
	v, n := binary.Uvarint(buf[6:])
	println(v, n)
	dec, err := hex.DecodeString("48656c6c6fzz")
	println(string(dec), err.Error())
	_, err = strconv.Atoi("12a")
	println(err.Error(), strconv.Quote("tab\t日\x00"), strconv.FormatInt(-255, 16))
	var bb bytes.Buffer
	bb.WriteString("hello ")
	bb.WriteRune('世')
	parts := bytes.Fields([]byte("  a b  c "))
	println(bb.String(), bb.Len(), len(parts), string(bytes.TrimSpace([]byte("  x  "))), bytes.Equal(nil, []byte{}))
}
