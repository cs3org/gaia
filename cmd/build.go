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
	"strings"

	"github.com/cs3org/gaia/pkg/builder"
	"github.com/spf13/cobra"
)

var buildFlags = struct {
	With           []string
	Output         string
	Debug          bool
	LeaveWorkspace bool
	BuildTags      []string
}{}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Create a custom build of reva",
	PreRunE: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		version := "latest"
		if len(args) != 0 {
			version = args[0]
		}

		plugins, replacement := parsePluginReplacement(buildFlags.With)
		builder := builder.Builder{
			RevaVersion:    version,
			Plugins:        plugins,
			Replacement:    replacement,
			Debug:          buildFlags.Debug,
			Log:            log,
			LeaveWorkspace: buildFlags.LeaveWorkspace,
			Tags:           buildFlags.BuildTags,
		}

		err := builder.Build(ctx, buildFlags.Output)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func parsePluginReplacement(l []string) ([]builder.Plugin, []builder.Replace) {
	var plugins []builder.Plugin
	var replacement []builder.Replace
	for _, e := range l {
		split := strings.SplitN(e, "=", 2)
		p := parsePlugin(split[0])
		plugins = append(plugins, p)

		if len(split) > 1 {
			replacement = append(replacement, parseReplace(p, split[1]))
		}
	}
	return plugins, replacement
}

func parsePlugin(s string) builder.Plugin {
	var p builder.Plugin
	split := strings.SplitN(s, "@", 2)
	p.RepositoryPath = split[0]
	if len(split) > 1 {
		p.Version = split[1]
	}
	return p
}

func parseReplace(p builder.Plugin, s string) builder.Replace {
	var r builder.Replace
	r.From = p.RepositoryPath
	split := strings.SplitN(s, "@", 2)
	r.To = split[0]
	if len(split) > 1 {
		r.ToVersion = split[1]
	}
	return r
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringSliceVar(&buildFlags.With, "with", nil, "plugins to include in the build")
	buildCmd.Flags().StringVarP(&buildFlags.Output, "output", "o", "./revad", "output file")
	buildCmd.Flags().BoolVarP(&buildFlags.Debug, "debug", "d", false, "compile with debug symbols")
	buildCmd.Flags().BoolVarP(&buildFlags.LeaveWorkspace, "leave-workspace", "l", false, "leave temporary build work space after execution")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildTags, "tags", nil, "list of additional build tags to consider satisfied during the build")
}
