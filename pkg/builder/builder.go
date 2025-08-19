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

package builder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/cs3org/gaia/internal/utils"
	"github.com/rs/zerolog"
)

const revaRepository = "github.com/cs3org/reva/v3"

type Platform struct {
	OS   string
	Arch string
}

type Builder struct {
	Platform
	RevaVersion    string
	Tags           []string
	Plugins        []Plugin
	Replacement    []Replace
	TempFolder     string
	Debug          bool
	Log            *zerolog.Logger
	LeaveWorkspace bool
	w              *workspace
}

func (b *Builder) getWorkspace() error {
	if b.Log == nil {
		log := zerolog.Nop()
		b.Log = &log
	}

	var err error

	if b.Platform.Arch == "" {
		b.Platform.Arch = utils.KeyFromGoEnv("GOARCH")
	}
	if b.Platform.OS == "" {
		b.Platform.OS = utils.KeyFromGoEnv("GOOS")
	}
	if b.RevaVersion == "" {
		b.RevaVersion = "latest"
	}

	b.w, err = b.newWorkspace()
	if err != nil {
		return err
	}

	b.w.setEnvKV("GOOS", b.Platform.OS)
	b.w.setEnvKV("GOARCH", b.Platform.Arch)

	return nil
}

func (b *Builder) Prepare(ctx context.Context) error {

	if b.w == nil {
		if err := b.getWorkspace(); err != nil {
			return err
		}
	}

	b.Log.Info().Msgf("preparing reva using version %s", b.RevaVersion)

	if err := b.w.runGoCommand(ctx, "mod", "init", "revad"); err != nil {
		return err
	}
	f, err := b.w.CreateFile("main.go")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := writeMainWithPlugins(f, b.Plugins); err != nil {
		return err
	}

	// if the reva repository has been replaced with a local one
	// it might have further replacements
	// if we do not consider them, the compilation will fail
	if path, ok := isRevaLocalReplacement(b.Replacement); ok {
		if gomod, err := parseGoModFile(ctx, filepath.Join(path, "go.mod")); err != nil {
			b.Log.Error().Err(err).Send()
		} else {
			for _, replace := range gomod.Replace {
				r := Replace{
					From:      replace.Old.Path,
					To:        replace.New.Path,
					ToVersion: replace.New.Version,
				}
				b.Replacement = append(b.Replacement, r)
				b.Log.Debug().Str("replace", r.Format()).Msg("inherited replace from local reva go.mod")
			}
		}
	}

	// do the replacement of the modules
	if len(b.Replacement) != 0 {
		if err := b.w.runGoModReplaceCommand(ctx, b.Replacement); err != nil {
			return err
		}
	}

	// TODO: verify all the versions
	for _, plugin := range b.Plugins {
		b.Log.Info().Msgf("adding plugin %s", plugin)
		if err := b.w.runGoGetCommand(ctx, plugin.RepositoryPath, plugin.Version); err != nil {
			return err
		}
	}
	if err := b.w.runGoGetCommand(ctx, revaRepository, b.RevaVersion); err != nil {
		return err
	}

	// run go mod tidy to fix all the modules
	if err := b.w.runGoCommand(ctx, "mod", "tidy"); err != nil {
		return err
	}

	// add compile time flags for version, commit, go version and build date
	// store them in the project so that it can be used independently
	bflags := b.w.generateBuildFlags(b.Replacement)
	f, err = b.w.CreateFile("bflags")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(bflags.Format())
	if err != nil {
		panic(err)
	}

	return nil
}

func (b *Builder) Build(ctx context.Context, output string) error {

	if b.w == nil {
		if err := b.getWorkspace(); err != nil {
			return err
		}
	}

	b.Log.Info().Msgf("building reva using workspace %s", b.w.folder)

	if output == "" {
		return errors.New("output file name cannot be empty")
	}
	var err error
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}

	args := make(buildArgs)
	if b.Debug {
		args.Add("-gcflags", "all=-N -l")
	} else {
		args.Add("-trimpath", "")
		args.Add("-ldflags", "-w", "-s")
	}
	if len(b.Tags) > 0 {
		args.Add("-tags", strings.Join(b.Tags, ","))
	}

	// add compile time flags for version, commit, go version and build date
	f, err := b.w.OpenFile("bflags")
	if err != nil {
		return err
	}
	defer f.Close()

	bflags, err := os.ReadFile(f.Name())
	if err != nil {
		panic(err)
	}

	b.Log.Debug().Interface("flags", bflags).Msg("using the following build flags")
	args.Add("-ldflags", string(bflags))

	b.Log.Info().Msg("building revad binary")
	if err := b.w.runGoBuildCommand(ctx, "main.go", output, args.Format()...); err != nil {
		return err
	}
	return nil
}

func (b *Builder) Close() {
	b.w.Close()
}
