// Copyright 2024 Google LLC
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

// Package importers provide utilities for importers.
package importers

import (
	"github.com/gx-org/gx/build/builder"
)

// Find finds an importer given a filter. Returns nil if the importer is not found.
func Find(importers []builder.Importer, filter func(imp builder.Importer) bool) builder.Importer {
	for _, imp := range importers {
		if filter(imp) {
			return imp
		}
	}
	return nil
}
