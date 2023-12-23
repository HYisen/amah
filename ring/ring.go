package ring

type Ring[T any] struct {
	items    []T
	begin    int
	end      int
	spacious bool
}

func New[T any](capacity int) Ring[T] {
	return Ring[T]{
		items:    make([]T, capacity),
		begin:    0,
		end:      0,
		spacious: true,
	}
}

func (r *Ring[T]) Empty() bool {
	return r.spacious && r.begin == r.end
}

// Get returns data stored in Ring.
// On 100k, cost 64us when it's full. 10k=6us 1k=1us 100=0.1us
// the major source seems to be the copied item count.
// the cost reduces relatively by occupied ratio if not full.
// and the cost reduces relatively by capacity ratio.
// There is also a simple version GetSimple,
// according to the benchmark in ring_test.go,
// the time cost differs from 2.34x to 7.23x,
// when cap and len ratio grows, this one excels more.
// BTW, tools/converter is the helper on benchmark report.
func (r *Ring[T]) Get() []T {
	size := len(r.items)
	if r.spacious {
		size = r.end
	}
	ret := make([]T, size)
	if r.end > r.begin {
		copy(ret, r.items[r.begin:r.end])
	} else if !r.Empty() {
		n := copy(ret, r.items[r.begin:])
		copy(ret[n:], r.items[:r.end])
	}
	return ret
}

func (r *Ring[T]) Add(item T) {
	if r.end < len(r.items) {
		if !r.spacious && r.begin == r.end {
			r.begin = (r.begin + 1) % len(r.items)
		}
		r.items[r.end] = item
		r.end++
	} else {
		if r.begin == 0 {
			r.begin = 1
		}
		r.items[0] = item
		r.end = 1
		r.spacious = false
	}
}
