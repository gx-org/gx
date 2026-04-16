// Copyright 2026 Google LLC
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

// Package proxies provides a way to distinguish proxy values for the interpreter from concrete values.
package proxies

import "github.com/gx-org/gx/build/ir"

// Proxy provides a function to determine if the value is a proxy or not.
// If the interface is not implemented, the value is considered to be a concrete value, that is not a proxy.
type Proxy interface {
	IsProxy() bool
}

// IsProxy returns true if the element is a proxy value or not.
func IsProxy(el ir.Element) bool {
	proxy, isProxy := el.(Proxy)
	if !isProxy {
		return false
	}
	return proxy.IsProxy()
}
