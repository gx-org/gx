package kernels

// Unary Operators

func boolNotArray(xVal Array) (Array, error) {
	x := toArray[bool](xVal)
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = !xi
	}
	return ToBoolArray(z, x.shape.AxisLengths), nil
}

func boolAndArray(xVal, yVal Array) (Array, error) {
	x, y := toArray[bool](xVal), toArray[bool](yVal)
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi && y.values[i]
	}
	return ToBoolArray(z, x.shape.AxisLengths), nil
}

func boolOrArray(xVal, yVal Array) (Array, error) {
	x, y := toArray[bool](xVal), toArray[bool](yVal)
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi || y.values[i]
	}
	return ToBoolArray(z, x.shape.AxisLengths), nil
}
