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

package platform

import (
	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// send raw data to the device. Return a handle from this package.
func (plat *Platform) send(dev compute.DeviceNum, data []byte, sh shapes.Shape) (*Handle, error) {
	if sh.DType == dtypes.InvalidDType {
		return nil, errors.Errorf("GX invalid data type not supported")
	}
	array, err := kernels.NewArrayFromRaw(data, sh)
	if err != nil {
		return nil, err
	}
	return NewDeviceHandle(plat, dev, array), nil
}
