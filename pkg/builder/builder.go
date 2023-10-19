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

	// do the replacement of the modules
	if len(b.Replacement) != 0 {
		if err := w.runGoModReplaceCommand(ctx, b.Replacement); err != nil {
			return err
		}
	}

	// run go mod tidy to fix all the modules
	if err := w.runGoCommand(ctx, "mod", "tidy"); err != nil {
		return err
	}

	buildArgs := []string{}
	if b.Debug {
		buildArgs = append(buildArgs, "-gcflags", "all=-N -l")
	} else {
		buildArgs = append(buildArgs, "-trimpath",
			"-ldflags", "-w -s") // trim debug symbols
	}
	// TODO: revad requires to set some compile time variables for setting the version
	b.Log.Info().Msg("building revad binary")
	if err := w.runGoBuildCommand(ctx, "main.go", output, buildArgs...); err != nil {
		return err
	}

	return nil
}

func writeMainWithPlugins(f *os.File, plugins []Plugin) error {
	return mainTemplate.Execute(f, plugins)
}

var mainTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	revadcmd "github.com/cs3org/reva/cmd/revad"
	{{ range . }}_ "{{ .RepositoryPath }}"
	{{ end }}
)

func main() {
	revadcmd.Main()
}
`))
