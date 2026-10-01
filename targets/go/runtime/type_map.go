package rt

import "reflect"

type mapEntry[K comparable, V any] struct {
	key   K
	value V
	live  bool
}

// Map is an insertion-ordered map with snapshot iteration. A nil *Map is
// the nil map.
type Map[K comparable, V any] struct {
	index   map[K]*mapEntry[K, V]
	entries []*mapEntry[K, V]
}

func (m *Map[K, V]) sourceMapTypes() (reflect.Type, reflect.Type) {
	return reflect.TypeFor[K](), reflect.TypeFor[V]()
}

// MapIter iterates a snapshot of the entries present when it was created.
type MapIter[K comparable, V any] struct {
	entries []*mapEntry[K, V]
	i       int
}

func (m *Map[K, V]) compact() {
	if len(m.entries) > 32 && len(m.entries) > 2*len(m.index) {
		live := make([]*mapEntry[K, V], 0, len(m.index))
		for _, e := range m.entries {
			if e.live {
				live = append(live, e)
			}
		}
		m.entries = live
	}
}
