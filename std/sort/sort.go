// Package sort sorts collections through the Len, Less, and Swap methods.
// Sort is stable here; Go only guarantees that Stable is.
package sort

// Interface is a collection that can be sorted by index.
type Interface interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
}

// Sort sorts data in ascending order as determined by Less.
func Sort(data Interface) { Stable(data) }

// Stable sorts data keeping the original order of equal elements.
func Stable(data Interface) {
	n := data.Len()
	const block = 20
	a := 0
	for ; a+block <= n; a += block {
		insertionSort(data, a, a+block)
	}
	insertionSort(data, a, n)
	for size := block; size < n; size *= 2 {
		for lo := 0; lo+size < n; lo += 2 * size {
			hi := lo + 2*size
			if hi > n {
				hi = n
			}
			symMerge(data, lo, lo+size, hi)
		}
	}
}

func insertionSort(data Interface, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

func symMerge(data Interface, a, m, b int) {
	if m-a == 1 {
		i, j := m, b
		for i < j {
			h := i + (j-i)/2
			if data.Less(h, a) {
				i = h + 1
			} else {
				j = h
			}
		}
		for k := a; k < i-1; k++ {
			data.Swap(k, k+1)
		}
		return
	}
	if b-m == 1 {
		i, j := a, m
		for i < j {
			h := i + (j-i)/2
			if !data.Less(m, h) {
				i = h + 1
			} else {
				j = h
			}
		}
		for k := m; k > i; k-- {
			data.Swap(k, k-1)
		}
		return
	}
	mid := a + (b-a)/2
	n := mid + m
	var start, r int
	if m > mid {
		start, r = n-b, mid
	} else {
		start, r = a, m
	}
	p := n - 1
	for start < r {
		c := start + (r-start)/2
		if !data.Less(p-c, c) {
			start = c + 1
		} else {
			r = c
		}
	}
	end := n - start
	if start < m && m < end {
		rotate(data, start, m, end)
	}
	if a < start && start < mid {
		symMerge(data, a, start, mid)
	}
	if mid < end && end < b {
		symMerge(data, mid, end, b)
	}
}

func swapRange(data Interface, a, b, n int) {
	for i := 0; i < n; i++ {
		data.Swap(a+i, b+i)
	}
}

func rotate(data Interface, a, m, b int) {
	i := m - a
	j := b - m
	for i != j {
		if i > j {
			swapRange(data, m-i, m, j)
			i -= j
		} else {
			swapRange(data, m-i, m+j-i, i)
			j -= i
		}
	}
	swapRange(data, m-i, m, i)
}

// IsSorted reports whether data is sorted.
func IsSorted(data Interface) bool {
	for i := data.Len() - 1; i > 0; i-- {
		if data.Less(i, i-1) {
			return false
		}
	}
	return true
}

type reverse struct {
	data Interface
}

func (r reverse) Len() int           { return r.data.Len() }
func (r reverse) Less(i, j int) bool { return r.data.Less(j, i) }
func (r reverse) Swap(i, j int)      { r.data.Swap(i, j) }

// Reverse returns data with the opposite order.
func Reverse(data Interface) Interface { return reverse{data} }

// IntSlice attaches the methods of Interface to []int.
type IntSlice []int

func (x IntSlice) Len() int           { return len(x) }
func (x IntSlice) Less(i, j int) bool { return x[i] < x[j] }
func (x IntSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// StringSlice attaches the methods of Interface to []string.
type StringSlice []string

func (x StringSlice) Len() int           { return len(x) }
func (x StringSlice) Less(i, j int) bool { return x[i] < x[j] }
func (x StringSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// Ints sorts x in increasing order.
func Ints(x []int) { Sort(IntSlice(x)) }

// Strings sorts x in increasing byte order.
func Strings(x []string) { Sort(StringSlice(x)) }

// IntsAreSorted reports whether x is sorted in increasing order.
func IntsAreSorted(x []int) bool { return IsSorted(IntSlice(x)) }

// StringsAreSorted reports whether x is sorted in increasing order.
func StringsAreSorted(x []string) bool { return IsSorted(StringSlice(x)) }

// Search returns the smallest index i in [0, n) for which f(i) is true,
// assuming f is false then true over the range, or n if there is none.
func Search(n int, f func(int) bool) int {
	i, j := 0, n
	for i < j {
		h := i + (j-i)/2
		if !f(h) {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// SearchInts returns the index at which x is or would be inserted in the
// sorted slice a.
func SearchInts(a []int, x int) int { return Search(len(a), func(i int) bool { return a[i] >= x }) }

// SearchStrings returns the index at which x is or would be inserted in the
// sorted slice a.
func SearchStrings(a []string, x string) int {
	return Search(len(a), func(i int) bool { return a[i] >= x })
}
