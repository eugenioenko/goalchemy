package lower

import "github.com/eugenioenko/goalchemy/internal/ir"

// prune removes blocks unreachable from the entry and renumbers the rest.
func prune(f *ir.Func) {
	if len(f.Blocks) == 0 {
		return
	}
	seen := map[*ir.Block]bool{}
	var order []*ir.Block
	var walk func(b *ir.Block)
	walk = func(b *ir.Block) {
		stack := []*ir.Block{b}
		for len(stack) > 0 {
			b := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[b] {
				continue
			}
			seen[b] = true
			order = append(order, b)
			succ := ir.Successors(b.Term)
			for i := len(succ) - 1; i >= 0; i-- {
				stack = append(stack, succ[i])
			}
		}
	}
	walk(f.Blocks[0])
	var kept []*ir.Block
	for _, b := range f.Blocks {
		if seen[b] {
			kept = append(kept, b)
		}
	}
	for i, b := range kept {
		b.ID = i
	}
	f.Blocks = kept
}
