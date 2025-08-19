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
	"fmt"
	"os"
	"strings"

	"github.com/cs3org/gaia/pkg/builder"
	"github.com/spf13/cobra"
)

var buildFlags = struct {
	With           []string
	Output         string
	Debug          bool
	LeaveWorkspace bool
	Workspace      string
	BuildTags      []string
	Vendor         bool
	OnlyPrepare    bool
	OnlyBuild      bool
}{}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Create a custom build of reva",
	PreRunE: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		if buildFlags.OnlyPrepare && buildFlags.OnlyBuild {
			fmt.Fprintln(os.Stderr, "Error: --only-prepare and --only-build cannot be used together")
			os.Exit(1)
		}

		if buildFlags.OnlyPrepare {
			buildFlags.LeaveWorkspace = true
		}

		if buildFlags.OnlyBuild && buildFlags.Workspace == "" {
			fmt.Fprintln(os.Stderr, "Error: asking to only build without specifying an existing workspace")
			os.Exit(2)
		}

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
			TempFolder:     buildFlags.Workspace,
			Tags:           buildFlags.BuildTags,
			Vendor:         buildFlags.Vendor,
		}
		defer builder.Close()

		if !buildFlags.OnlyBuild {
			err := builder.Prepare(ctx)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}

		if !buildFlags.OnlyPrepare {
			err := builder.Build(ctx, buildFlags.Output)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
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
	buildCmd.Flags().StringVarP(&buildFlags.Workspace, "workspace", "w", "", "path where to create the build files, leave empty for temp folder")
	buildCmd.Flags().StringSliceVar(&buildFlags.BuildTags, "tags", nil, "list of additional build tags to consider satisfied during the build")
	buildCmd.Flags().BoolVarP(&buildFlags.Vendor, "vendor", "", false, "uses vendoring to keep all dependencies local")
	buildCmd.Flags().BoolVarP(&buildFlags.OnlyPrepare, "only-prepare", "", false, "only run the prepare workspace stage (forces --leave-workspace)")
	buildCmd.Flags().BoolVarP(&buildFlags.OnlyBuild, "only-build", "", false, "only run the build workspace stage (requires --workspace to be set)")
}
