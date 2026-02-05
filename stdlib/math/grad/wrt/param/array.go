// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package param

import "github.com/gx-org/gx/stdlib/math/grad/wrt"

// Array is a parameter in a function with the array type.
type Array struct {
	core
	WRT *wrt.WithRespectTo
}

// Arrays returns the list of array for which the gradient needs to be computed with respect to.
// In this case, it just returns itself.
func (p *Array) Arrays() []*Array {
	return []*Array{p}
}
