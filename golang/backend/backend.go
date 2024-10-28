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

// Package backend implements a backend for GX in Go.
package backend

import (
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	gograph "github.com/gx-org/gx/golang/backend/graph"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
)

type backend struct {
	plat *goplatform.Platform
}

// New native Go runtime for GX.
func New(builder *builder.Builder) *api.Runtime {
	return api.NewRuntime(&backend{
		plat: goplatform.New(),
	}, builder)
}

// Platform returns the Go native platform.
func (bck *backend) Platform() platform.Platform {
	return bck.plat
}

// NewGraph returns a graph builder for native Go operations.
func (bck *backend) NewGraph(funcName string) graph.Graph {
	return gograph.New(bck.plat, funcName)
}
