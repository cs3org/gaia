// Copyright 2018-2023 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var log *zerolog.Logger

var globalFlags = struct {
	Verbose int
}{}

var rootCmd = &cobra.Command{
	Use:   "gaia",
	Short: "A reva builder",
	Long:  "A tool to make it easy to create custom build of reva.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().CountVarP(&globalFlags.Verbose, "verbose", "v", "verbosity")
	cobra.OnInitialize(func() {
		initLogger(globalFlags.Verbose)
	})
}

func initLogger(verbosity int) {
	level := zerolog.InfoLevel
	if verbosity > 0 {
		level = zerolog.DebugLevel
	}
	out := zerolog.ConsoleWriter{Out: os.Stderr}
	l := zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger().
		Level(level).
		Output(out)
	log = &l
}
