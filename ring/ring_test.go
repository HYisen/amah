package ring

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRingLoop(t *testing.T) {
	r := New[int](3)
	type step struct {
		name string
		add  bool
		neo  int
		want []int
	}
	steps := []step{
		{"empty", false, 0, nil},
		{"+1", true, 1, []int{1}},
		{"stable", false, 0, []int{1}},
		{"+2", true, 2, []int{1, 2}},
		{"+3 full", true, 3, []int{1, 2, 3}},
		{"+4 overlap", true, 4, []int{2, 3, 4}},
		{"+5 166%", true, 5, []int{3, 4, 5}},
		{"+6 200%", true, 6, []int{4, 5, 6}},
	}
	for _, s := range steps {
		if s.add {
			r.Add(s.neo)
		}
		got := r.Get()
		if !reflect.DeepEqual(got, s.want) {
			t.Errorf("on %s, want %v got %v", s.name, s.want, got)
		}
	}
}

func newRing(len, cap int) Ring[int] {
	r := New[int](cap)
	for i := 0; i < len; i++ {
		r.Add(i)
	}
	return r
}

func BenchmarkRing_Get(b *testing.B) {
	nameToCap := map[string]int{
		"10":   10,
		"100":  100,
		"1k":   1000,
		"10k":  10_000,
		"100k": 100_000,
	}
	nameToLenFunc := map[string]func(cap int) (len int){
		"few": func(_ int) (len int) {
			return 5
		},
		"25%": func(cap int) (len int) {
			return max(1, cap/4)
		},
		"50%": func(cap int) (len int) {
			return max(1, cap/2)
		},
		"100%": func(cap int) (len int) {
			return cap
		},
		"125%": func(cap int) (len int) {
			return cap + cap/4
		},
		"150%": func(cap int) (len int) {
			return cap + cap/2
		},
		"175%": func(cap int) (len int) {
			return cap*2 - cap/4
		},
	}
	for capName, capacity := range nameToCap {
		for lenName, lenFunc := range nameToLenFunc {
			r := newRing(lenFunc(capacity), capacity)
			b.Run(fmt.Sprintf("v0 %s %s", capName, lenName), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					r.GetSimple()
				}
			})
			b.Run(fmt.Sprintf("v1 %s %s", capName, lenName), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					r.Get()
				}
			})
		}
	}
}

func (r *Ring[T]) GetSimple() []T {
	var ret []T
	if r.end > r.begin {
		for i := r.begin; i < r.end; i++ {
			ret = append(ret, r.items[i])
		}
	} else if !r.Empty() {
		for i := r.begin; i < len(r.items); i++ {
			ret = append(ret, r.items[i])
		}
		for i := 0; i < r.end; i++ {
			ret = append(ret, r.items[i])
		}
	}
	return ret
}

func BenchmarkRing_Add(b *testing.B) {
	nameToCap := map[string]int{
		"10":   10,
		"100":  100,
		"1k":   1000,
		"10k":  10_000,
		"100k": 100_000,
	}
	for name, capacity := range nameToCap {
		r := New[int](capacity)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r.Add(i)
			}
		})
	}
}
