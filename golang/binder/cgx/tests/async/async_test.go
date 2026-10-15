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

package async_test

import (
	"testing"

	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/binder/cgx/tests/async"
	"github.com/gx-org/gx/golang/binder/cgx/tests/gocgx"
)

func TestAsyncCGXGoBackend(t *testing.T) {
	rtm, err := backend.New(gocgx.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	defer rtm.Backend().Finalize()
	async.RunTestAsyncCGX(t, rtm, async.RunFunctionReturnFloat32)
}
