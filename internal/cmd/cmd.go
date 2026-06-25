// Copyright 2026 Google LLC
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

// Package cmd is the root package for all the commands of the GX command.
package cmd

import (
	"google3/third_party/golang/cobra/cobra"
	"github.com/gx-org/gx/internal/cmd/debug"
	"github.com/gx-org/gx/internal/cmd/impgo"
)

var rootCmd = &cobra.Command{
	Use:           "gx",
	Short:         "GX language command",
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	rootCmd.SetErrPrefix("gx:")
	rootCmd.PersistentFlags().BoolVarP(&debug.Debug, "debug", "d", false, "print debug information")
	rootCmd.AddCommand(impgo.Cmd())
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
