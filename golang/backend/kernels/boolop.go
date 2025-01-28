package kernels

// Unary Operators

func boolNotArray(xVal Array) (Array, error) {
	x := xVal.(*ArrayT[bool])
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = !xi
	}
	return toBoolArray(z, x.shape.AxisLengths), nil
}
