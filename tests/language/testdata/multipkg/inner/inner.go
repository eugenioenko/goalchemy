package inner

import "goalchemy/tests/language/testdata/multipkg/inner/deep"

var Value = deep.Base + 1

var count int

func init() {
	count = deep.Base * 10
	println("inner init", Value)
}

func Count() int { return count }

type Counter struct{ total int }

func NewCounter(start int) *Counter { return &Counter{start} }

func (c *Counter) Add(n int)  { c.total += n }
func (c *Counter) Total() int { return c.total }

type Shape interface{ Area() int }

type Square struct{ Side int }

func (s Square) Area() int { return s.Side * s.Side }

func Describe(c *Counter) string {
	if c.total > 5 {
		return "big"
	}
	return "small"
}
