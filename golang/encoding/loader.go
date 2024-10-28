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

package encoding

import (
	"sync"

	"go.uber.org/multierr"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

// numWorkers is the number of simultaneous workers loading data.
const numWorkers = 16

type asyncErrors struct {
	locker sync.Mutex
	errs   error
}

func (ae *asyncErrors) add(err error) {
	ae.locker.Lock()
	defer ae.locker.Unlock()

	ae.errs = multierr.Append(ae.errs, err)
}

func (ae *asyncErrors) errors() error {
	ae.locker.Lock()
	defer ae.locker.Unlock()

	errs := ae.errs
	ae.errs = nil
	return errs
}

type (
	leafToFieldInfo struct {
		setter func(types.Bridge) error
		future ValueFuture
	}

	// loader loads leaf values asynchronously in Go routines.
	loader struct {
		wg       sync.WaitGroup
		errs     asyncErrors
		toWorker chan leafToFieldInfo
	}
)

func newLoader(bProc BridgeProcessor) *loader {
	ld := &loader{
		toWorker: make(chan leafToFieldInfo),
	}
	for range numWorkers {
		ld.wg.Add(1)
		go ld.setFieldWithLeafWorker(bProc)
	}
	return ld
}

func (ld *loader) setNumerical(data Data, setter func(types.Bridge) error) error {
	future, err := data.ValueFuture()
	if err != nil {
		return err
	}
	ld.toWorker <- leafToFieldInfo{
		setter: setter,
		future: future,
	}
	return nil
}

func (info leafToFieldInfo) process(bProc BridgeProcessor) error {
	val, err := info.future.Value()
	if err != nil {
		return err
	}
	if bProc == nil {
		goto setField
	}
	if val, err = bProc(val); err != nil {
		return err
	}
setField:
	return info.setter(val)
}

func (ld *loader) setFieldWithLeafWorker(bProc BridgeProcessor) {
	defer ld.wg.Done()
	for info := range ld.toWorker {
		if err := info.process(bProc); err != nil {
			ld.errs.add(err)
		}
	}
}

func (ld *loader) close() error {
	close(ld.toWorker)
	ld.wg.Wait()
	return ld.errs.errors()
}
