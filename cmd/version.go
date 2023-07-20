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
	"runtime/debug"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Println(version())
	},
}

func version() string {
	mod := getModule()
	ver := mod.Version
	if mod.Replace != nil {
		ver += " => " + mod.Replace.Path
		if mod.Replace.Version != "" {
			ver += "@" + mod.Replace.Version
		}
	}
	return ver
}

func getModule() *debug.Module {
	mod := &debug.Module{}
	mod.Version = "<unknown>"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return mod
	}
	for _, m := range bi.Deps {
		if m.Path == "github.com/cs3org/gaia" {
			return m
		}
	}
	return mod
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
