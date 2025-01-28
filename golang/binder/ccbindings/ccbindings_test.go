package ccbindings_test

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/binder/ccbindings"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var tests = []struct {
	path    string
	hWants  []string
	ccWants []string
}{
	{
		path: "github.com/gx-org/gx/tests/bindings/dtypes",
		ccWants: []string{
			"DtypesIR::Load",
			"Dtypes::BuildFor",
		},
		hWants: []string{
			"class DtypesIR",
			"class Dtypes",
		},
	},
}

func checkGenerate(t *testing.T, gen func(io.Writer) error, wants []string) {
	out := &strings.Builder{}
	if err := gen(out); err != nil {
		t.Fatalf("cannot generate bindings:\n%+v\n\nOutput generated:\n%s", err, gxtesting.NumberLines(out.String()))
	}
	got := out.String()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("%q cannot be found in generated bindings:\n%s", want, got)
		}
	}
}

func run(t *testing.T, bld *builder.Builder, path string, hWants, ccWants []string) {
	pkg, err := bld.Build(path)
	if err != nil {
		t.Errorf("cannot get package %s builder: %v", path, err)
		return
	}
	bnd, err := ccbindings.New(pkg.IR())
	if err != nil {
		t.Fatal(err)
	}
	files := bnd.Files()
	hFile, ccFile := files[0], files[1]
	checkGenerate(t, hFile.WriteBindings, hWants)
	checkGenerate(t, ccFile.WriteBindings, ccWants)
}

func TestCCBindings(t *testing.T) {
	builder := gxtesting.NewBuilderStaticSource(nil)
	for _, test := range tests {
		_, name := filepath.Split(test.path)
		t.Run(name, func(t *testing.T) {
			run(t, builder, test.path, test.hWants, test.ccWants)
		})
	}
}
