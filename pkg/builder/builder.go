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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/cs3org/gaia/internal/utils"
	"github.com/rs/zerolog"
)

const revaRepository = "github.com/cs3org/reva"

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
}

type Plugin struct {
	RepositoryPath string
	Version        string
}

func (p Plugin) String() string {
	s := p.RepositoryPath
	if p.Version != "" {
		s += "@" + p.Version
	}
	return s
}

type Replace struct {
	From      string
	To        string
	ToVersion string
}

// Format formats the replacement string to be valid
// with the go mod edit command.
func (r Replace) Format() string {
	s := r.From + "=" + r.To
	if r.ToVersion != "" {
		s += "@" + r.ToVersion
	}
	return s
}

func (r Replace) String() string {
	s := r.From + " => " + r.To
	if r.ToVersion != "" {
		s += "@" + r.ToVersion
	}
	return s
}

func (b *Builder) Build(ctx context.Context, output string) error {
	if output == "" {
		return errors.New("output file name cannot be empty")
	}
	if b.Log == nil {
		log := zerolog.Nop()
		b.Log = &log
	}

	b.Log.Info().Msgf("building reva using version %s", b.RevaVersion)

	var err error
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}

	if b.Platform.Arch == "" {
		b.Platform.Arch = utils.KeyFromGoEnv("GOARCH")
	}
	if b.Platform.OS == "" {
		b.Platform.OS = utils.KeyFromGoEnv("GOOS")
	}
	if b.RevaVersion == "" {
		b.RevaVersion = "latest"
	}

	w, err := b.newWorkspace()
	if err != nil {
		return err
	}
	defer w.Close()

	w.setEnvKV("GOOS", b.Platform.OS)
	w.setEnvKV("GOARCH", b.Platform.Arch)

	if err := w.runGoCommand(ctx, "mod", "init", "revad"); err != nil {
		return err
	}
	f, err := w.CreateFile("main.go")
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
		if err := w.runGoModReplaceCommand(ctx, b.Replacement); err != nil {
			return err
		}
	}

	// TODO: verify all the versions
	for _, plugin := range b.Plugins {
		b.Log.Info().Msgf("adding plugin %s", plugin)
		if err := w.runGoGetCommand(ctx, plugin.RepositoryPath, plugin.Version); err != nil {
			return err
		}
	}
	if err := w.runGoGetCommand(ctx, revaRepository, b.RevaVersion); err != nil {
		return err
	}

	// run go mod tidy to fix all the modules
	if err := w.runGoCommand(ctx, "mod", "tidy"); err != nil {
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
	bflags := w.generateBuildFlags(b.Replacement)
	b.Log.Debug().Interface("flags", bflags).Msg("using the following build flags")
	args.Add("-ldflags", bflags.Format())

	b.Log.Info().Msg("building revad binary")
	if err := w.runGoBuildCommand(ctx, "main.go", output, args.Format()...); err != nil {
		return err
	}

	return nil
}

func writeMainWithPlugins(f *os.File, plugins []Plugin) error {
	plugins = slices.DeleteFunc(plugins, func(p Plugin) bool { return p.RepositoryPath == revaRepository })
	return mainTemplate.Execute(f, struct {
		Plugins  []Plugin
		RevaRepo string
	}{
		Plugins:  plugins,
		RevaRepo: revaRepository,
	})
}

type GoMod struct {
	Module struct {
		Path string `json:"Path"`
	} `json:"Module"`
	Go      string `json:"Go"`
	Require []struct {
		Path     string `json:"Path"`
		Version  string `json:"Version"`
		Indirect bool   `json:"Indirect,omitempty"`
	} `json:"Require"`
	Exclude any `json:"Exclude"`
	Replace []struct {
		Old struct {
			Path string `json:"Path"`
		} `json:"Old"`
		New struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
		} `json:"New"`
	} `json:"Replace"`
	Retract any `json:"Retract"`
}

func parseGoModFile(ctx context.Context, path string) (*GoMod, error) {
	var stdout bytes.Buffer

	c := exec.CommandContext(ctx, utils.Go(), "mod", "edit", "-json", path)
	c.Stdout = &stdout

	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("error running command: %w", err)
	}

	var gomod GoMod
	if err := json.NewDecoder(&stdout).Decode(&gomod); err != nil {
		return nil, fmt.Errorf("error decoding go.mod: %w", err)
	}
	return &gomod, nil
}

func isRevaLocalReplacement(repl []Replace) (string, bool) {
	for _, r := range repl {
		if r.From == revaRepository {
			_, err := os.Stat(r.To)
			return r.To, err == nil
		}
	}
	return "", false
}

var mainTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	revadcmd "{{.RevaRepo}}/cmd/revad"
{{- range .Plugins }}
	_ "{{ .RepositoryPath }}"
{{- end }}
)

func main() {
	revadcmd.Main()
}
`))
