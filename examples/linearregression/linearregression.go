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

// Package linearregressionexperiment runs the linear regression example given a backend.
package linearregressionexperiment

import (
	"fmt"
	"math"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/examples/linearregression/linearregression_go_gx"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

// Run the linear regression example for the package ready to be compiled for a device.
func Run(rtm *api.Runtime, numSteps int) error {
	dev, err := rtm.Device(0)
	if err != nil {
		return err
	}
	// Build the GX code to the first device on the given backend.
	linearregression, err := linearregression_go_gx.BuildFor(
		dev,
		linearregression_go_gx.Size.Set(1e6),
	)
	if err != nil {
		return err
	}
	// Create a new target on the device given a seed.
	// The target generates samples that the learner learns from.
	seed := types.Int64(0)
	bias := types.Float32(5)
	target, err := linearregression.NewTarget(seed, bias)
	if err != nil {
		return err
	}
	// Create a new learner that will update its weights towards the target given generated samples.
	stepSize := types.Float32(1e-6)
	learner, err := linearregression.NewLearner(stepSize)
	if err != nil {
		return err
	}
	// Main experiment loop
	for range numSteps {
		// Generate a sample from the target.
		var features types.Array[float32]
		var targetValue types.Atom[float32]
		target, features, targetValue, err = target.Sample()
		if err != nil {
			return err
		}
		// Update the learner with the sample.
		// Note that features and targetValue stays on the device
		// and are not transferred to the host.
		var predictionErrorDevice types.Atom[float32]
		learner, predictionErrorDevice, err = learner.Update(features, targetValue)
		if err != nil {
			return err
		}
		predictionError, err := predictionErrorDevice.FetchValue()
		if err != nil {
			return err
		}
		const roundAt = 1000
		predictionErrorFloored := math.Ceil(float64(predictionError)/roundAt) * roundAt
		fmt.Println("predictionError <", predictionErrorFloored)
	}
	return nil
}
