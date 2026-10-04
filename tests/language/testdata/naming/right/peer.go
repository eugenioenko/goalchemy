package peer

type PolicyBinding struct {
	Algorithm    string
	Hash         string
	LegacyString bool
}
type Counter struct{ Count int }

func (c Counter) Value() int { return c.Count + 2 }
func New(n int) Counter      { return Counter{n} }
