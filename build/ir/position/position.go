// Package position locates GX IR objects by source cursor position.
package position

import (
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

// At returns the most specific IR object in file spanning the 1-based line and
// byte column, or nil if no IR object covers that position.
func At(file *ir.File, line, column int) ir.IR {
	if file == nil || line < 1 || column < 1 {
		return nil
	}
	tokFile := file.FileSet().File(file.Src.Pos())
	if line > tokFile.LineCount() {
		return nil
	}
	lineOffset := tokFile.Offset(tokFile.LineStart(line))
	if column > tokFile.Size()-lineOffset {
		return nil
	}
	target := tokFile.Pos(lineOffset + column - 1)
	if tokFile.Line(target) != line {
		return nil
	}

	f := &finder{target: target}
	file.Visit(f)
	return f.best
}

type finder struct {
	target   token.Pos
	best     ir.IR
	bestSpan token.Pos
}

var _ ir.Finder = (*finder)(nil)

func (f *finder) Consider(node ir.IR, start, end token.Pos) bool {
	if !start.IsValid() || f.target < start || f.target >= end {
		return false
	}
	span := end - start
	if f.best == nil || span <= f.bestSpan {
		f.best = node
		f.bestSpan = span
	}
	return true
}
