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
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	// Binary is the program to build. Empty is revad.
	Binary Binary
	// Brand is the brand file of a reva build.
	Brand          string
	RevaVersion    string
	Tags           []string
	Plugins        []Plugin
	Replacement    []Replace
	TempFolder     string
	Debug          bool
	Log            *zerolog.Logger
	LeaveWorkspace bool
	Vendor         bool
	LdFlags        string
	Static         bool
	StaticMusl     bool
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

	if b.StaticMusl {
		if err := b.checkMuslGcc(); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) checkMuslGcc() error {
	cmd := exec.Command("sh", "-c", "command -v musl-gcc")
	if err := cmd.Run(); err != nil {
		return errors.New("musl-gcc not found. Install with: sudo apt install musl-tools (or equivalent for your distribution)")
	}
	return nil
}

func (b *Builder) Prepare(ctx context.Context) error {

	if b.w == nil {
		if err := b.getWorkspace(); err != nil {
			return err
		}
	}

	binary, err := ParseBinary(string(b.Binary))
	if err != nil {
		return err
	}

	var brand []byte
	if b.Brand != "" {
		if binary != Reva {
			return fmt.Errorf("a brand is for the reva binary, not %s", binary)
		}
		if brand, err = os.ReadFile(b.Brand); err != nil {
			return fmt.Errorf("error reading the brand: %w", err)
		}
	}

	b.Log.Info().Msgf("preparing %s using reva version %s", binary, b.RevaVersion)

	if err := b.w.runGoCommand(ctx, "mod", "init", string(binary)); err != nil {
		return err
	}
	f, err := b.w.CreateFile("main.go")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := writeMain(f, binary, b.Plugins, brand != nil); err != nil {
		return err
	}

	// main.go embeds the brand, so the workspace needs its own copy.
	if brand != nil {
		b.Log.Info().Msgf("using brand %s", b.Brand)
		if err := b.w.WriteFile(brandFile, brand); err != nil {
			return err
		}
	}

	b.Replacement, err = resolveReplacements(ctx, b.Replacement)
	if err != nil {
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

	if b.Vendor {
		// Keep all modules locally
		if err := b.w.runGoCommand(ctx, "mod", "vendor"); err != nil {
			return err
		}
	}

	if binary == Reva {
		if err := b.checkReva(ctx); err != nil {
			return err
		}
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

	if b.StaticMusl {
		b.w.setEnvKV("CC", "musl-gcc")
		b.Log.Info().Msg("using musl-gcc for static build")
	}

	args := b.compileArgs()

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

	ldflags := string(bflags)

	if b.LdFlags != "" {
		if ldflags != "" {
			ldflags += " "
		}
		ldflags += b.LdFlags
		b.Log.Info().Str("ldflags", b.LdFlags).Msg("adding custom ldflags")
	}

	if b.StaticMusl {
		if !strings.Contains(ldflags, "-extldflags") {
			if ldflags != "" {
				ldflags += " "
			}
			ldflags += "-extldflags '-static'"
			b.Log.Info().Msg("adding musl static linking flags (-extldflags '-static')")
		} else {
			b.Log.Info().Msg("extldflags already present in ldflags, skipping")
		}
	} else if b.Static {
		if !strings.Contains(ldflags, "-extldflags=-static") {
			if ldflags != "" {
				ldflags += " "
			}
			ldflags += "-extldflags=-static"
			b.Log.Info().Msg("adding static linking flags (-extldflags=-static)")
		} else {
			b.Log.Info().Msg("static linking flag already present in ldflags, skipping")
		}
	}

	args.Add("-ldflags", ldflags)

	b.Log.Info().Msgf("building %s binary", cmp.Or(b.Binary, Revad))
	return b.w.runGoBuildCommand(ctx, "main.go", output, args.Format()...)
}

func (b *Builder) compileArgs() buildArgs {
	args := make(buildArgs)
	if b.Debug {
		args.Add("-gcflags", "all=-N -l")
	} else {
		args.Add("-trimpath", "")
		args.Add("-ldflags", "-w", "-s")
	}

	if b.StaticMusl {
		tags := make([]string, 0, len(b.Tags)+1)
		tags = append(tags, b.Tags...)
		if !slices.Contains(tags, "sqlite_omit_load_extension") {
			tags = append(tags, "sqlite_omit_load_extension")
			b.Log.Info().Msg("adding sqlite_omit_load_extension build tag for musl static build")
		}
		args.Add("-tags", strings.Join(tags, ","))
	} else if len(b.Tags) > 0 {
		args.Add("-tags", strings.Join(b.Tags, ","))
	}

	if b.Vendor {
		args.Add("-mod=vendor")
	}
	return args
}

// checkReva makes the reva CLI check its brand and plugins. gaia does not
// know the schema of the brand. The check runs on the host, also when the
// build is for another platform.
func (b *Builder) checkReva(ctx context.Context) error {
	b.Log.Info().Msg("checking the brand and the plugins of the build")
	args := append([]string{"run"}, b.compileArgs().Format()...)
	args = append(args, ".", brandCheckArg)
	host := []string{"GOOS=" + utils.KeyFromGoEnv("GOHOSTOS"), "GOARCH=" + utils.KeyFromGoEnv("GOHOSTARCH")}
	if err := b.w.runGoCommandEnv(ctx, host, args...); err != nil {
		return fmt.Errorf("error checking the reva build (go run . %s): %w", brandCheckArg, err)
	}
	return nil
}

func (b *Builder) Close() {
	b.w.Close()
}
