package ordered_test

import (
	"testing"

	"github.com/gx-org/gx/base/ordered"
)

type entry struct {
	k string
	v int
}

func TestMap(t *testing.T) {
	tests := []struct {
		entries []entry
		want    []entry
	}{
		{
			entries: []entry{
				{k: "a", v: 1},
				{k: "b", v: 2},
				{k: "c", v: 3},
			},
			want: []entry{
				{k: "a", v: 1},
				{k: "b", v: 2},
				{k: "c", v: 3},
			},
		},
		{
			entries: []entry{
				{k: "a", v: 1},
				{k: "b", v: 2},
				{k: "a", v: 3},
			},
			want: []entry{
				{k: "a", v: 3},
				{k: "b", v: 2},
			},
		},
		{
			entries: []entry{
				{k: "a", v: 1},
				{k: "a", v: 2},
				{k: "a", v: 3},
				{k: "a", v: 4},
			},
			want: []entry{
				{k: "a", v: 4},
			},
		},
	}
	for ti, test := range tests {
		m := ordered.NewMap[string, int]()
		for _, entry := range test.entries {
			m.Store(entry.k, entry.v)
		}
		if m.Size() != len(test.want) {
			t.Errorf("test %d: map has %d entries but want %d", ti, m.Size(), len(test.want))
			continue
		}

		// Clone the map before the tests.
		m = m.Clone()

		// Iterate from the key.
		i := 0
		for gotK := range m.Keys() {
			gotV, _ := m.Load(gotK)
			wantK, wantV := test.want[i].k, test.want[i].v
			if gotV != wantV {
				t.Errorf("test %d entry %d: got %s->%d but want %s->%d", ti, i, gotK, gotV, wantK, wantV)
			}
			i++
		}

		// Iterate over all the items.
		i = 0
		for gotK, gotV := range m.Iter() {
			wantK, wantV := test.want[i].k, test.want[i].v
			if gotK != wantK || gotV != wantV {
				t.Errorf("test %d entry %d: got %s->%d but want %s->%d", ti, i, gotK, gotV, wantK, wantV)
			}
			i++
		}

		// Iterate over all the values.
		i = 0
		for gotV := range m.Values() {
			wantK, wantV := test.want[i].k, test.want[i].v
			if gotV != wantV {
				t.Errorf("test %d entry %d: got .->%d but want %s->%d", ti, i, gotV, wantK, wantV)
			}
			i++
		}
	}
}
