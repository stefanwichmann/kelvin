// MIT License
//
// # Copyright (c) 2019 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"testing"
)

func TestEqualsFloat(t *testing.T) {
	// Should be equal
	equal := equalsFloat([]float32{0, 0}, []float32{0, 0}, 0)
	if !equal {
		t.Errorf("equalsFloat([]float32{0, 0}, []float32{0, 0}, 0) = %t; want true", equal)
	}

	equal = equalsFloat([]float32{-1, -1}, []float32{-1, -1}, 0)
	if !equal {
		t.Errorf("equalsFloat([]float32{-1, -1}, []float32{-1, -1}, 0) = %t; want true", equal)
	}

	equal = equalsFloat([]float32{1.001, -1.001}, []float32{1.001, -1.001}, 0)
	if !equal {
		t.Errorf("equalsFloat([]float32{1.001, -1.001}, []float32{1.001, -1.001}, 0) = %t; want true", equal)
	}

	equal = equalsFloat([]float32{1.0, 0}, []float32{1.001, 0}, 0.001)
	if !equal {
		t.Errorf("equalsFloat([]float32{1.0, 0}, []float32{1.001, 0}, 0.001) = %t; want true", equal)
	}

	// Should not be equal
	equal = equalsFloat([]float32{0.5, 0.5}, []float32{-1, -1}, 0)
	if equal {
		t.Errorf("equalsFloat([]float32{0.5, 0.5}, []float32{-1, -1}, 0) = %t; want false", equal)
	}

	equal = equalsFloat([]float32{-1}, []float32{-1, -1}, 0)
	if equal {
		t.Errorf("equalsFloat([]float32{-1}, []float32{-1, -1}, 0) = %t; want false", equal)
	}

	equal = equalsFloat([]float32{1.0, 0}, []float32{1.002, 0}, 0.001)
	if equal {
		t.Errorf("equalsFloat([]float32{1.0, 0}, []float32{1.002, 0}, 0.001) = %t; want false", equal)
	}
}
