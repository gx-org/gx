package fmt_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	gxfmt "github.com/gx-org/gx/base/fmt"
)

func TestNumber(t *testing.T) {
	tests := []struct {
		txt  string
		want string
	}{
		{
			txt: `
Hello
World
`,
			want: `
1 Hello
2 World
`,
		},
		{
			txt: `
Line1
Line2
Line3
Line4
Line5
Line6
Line7
Line8
Line9
Line10
`,
			want: `
01 Line1
02 Line2
03 Line3
04 Line4
05 Line5
06 Line6
07 Line7
08 Line8
09 Line9
10 Line10
`,
		},
	}
	for _, test := range tests {
		got := gxfmt.Number(strings.TrimSpace(test.txt))
		want := strings.TrimSpace(test.want)
		if got != want {
			t.Errorf("got:\n%s\nbut want:\n%s\ndiff:\n%s", got, want, cmp.Diff(got, want))
		}
	}
}
