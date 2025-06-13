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

// Package prime generates prime numbers.
package prime

import "fmt"

// Prime generates prime numbers.
type Prime struct {
	min       int
	generated []int
}

// New returns a new prime generator given a minimum.
func New(min int) *Prime {
	if min < 0 {
		panic(fmt.Sprintf("got minimum prime number %d, want >= 0", min))
	}
	return &Prime{min: min}
}

func (p *Prime) isDivisable(val int) bool {
	for _, v := range p.generated[1:] {
		if val%v == 0 {
			return true
		}
	}
	return false
}

func (p *Prime) findNext() int {
	if len(p.generated) == 0 {
		return 2
	}
	if len(p.generated) == 1 {
		return 3
	}
	last := p.generated[len(p.generated)-1]
	for i := 2; ; i += 2 {
		val := last + i
		if p.isDivisable(val) {
			continue
		}
		return val
	}
}

// Next returns the next prime number.
func (p *Prime) Next() int {
	next := -1
	for next < p.min {
		next = p.findNext()
		p.generated = append(p.generated, next)
		if next >= p.min {
			return next
		}
	}
	return -1
}
