// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handle

import (
	"runtime"
	"unsafe"

	"github.com/gx-org/gx/base/sync"
)

/* pinners of slice data */

// pinners holds runtime.Pinners that are used to temporarily pin data in place so it becomes
// accessible to C.
var pinners = sync.Map[unsafe.Pointer, *runtime.Pinner]{}

// PinSliceData pins the content of a slice so that it can shared with C.
// Call unpinSliceData to release the reference to the slice data.
func PinSliceData[T any](vs []T) unsafe.Pointer {
	if vs == nil {
		return nil
	}
	ptr := unsafe.Pointer(unsafe.SliceData(vs))
	pinner := runtime.Pinner{}
	pinner.Pin(ptr)
	pinners.Store(ptr, &pinner)
	return ptr
}

// UnpinSliceData unpin the data of a slice.
func UnpinSliceData(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	pinner := pinners.Load(ptr)
	pinner.Unpin()
	pinners.Delete(ptr)
}
