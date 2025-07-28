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
	"encoding/json"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/cs3org/gaia/internal/utils"
)

type buildArgs map[string][]string

func (b *buildArgs) Add(key string, val ...string) {
	(*b)[key] = append((*b)[key], val...)
}

func formatOptionArgument(vals []string) string {
	return strings.Join(slices.DeleteFunc(vals, func(v string) bool { return v == "" }), " ")
}

func (b buildArgs) Format() (args []string) {
	for k, vals := range b {
		args = append(args, k)
		if optArgs := formatOptionArgument(vals); optArgs != "" {
			args = append(args, optArgs)
		}
	}
	return
}

type buildFlags struct {
	GitCommit string
	Version   string
	GoVersion string
	BuildDate time.Time
}

const buildVariablesPkg = revaRepository + "/cmd/revad"

func generateBuildFlag(key, val string) string {
	return "-X " + key + "=" + val
}

func (b buildFlags) Format() string {
	var params []string
	if b.GitCommit != "" {
		params = append(params, generateBuildFlag(buildVariablesPkg+".gitCommit", b.GitCommit))
	}
	if b.Version != "" {
		params = append(params, generateBuildFlag(buildVariablesPkg+".version", b.Version))
	}
	if b.GoVersion != "" {
		params = append(params, generateBuildFlag(buildVariablesPkg+".goVersion", b.GoVersion))
	}
	if b.BuildDate.Unix() != 0 {
		params = append(params, generateBuildFlag(buildVariablesPkg+".buildDate", b.BuildDate.Format(time.RFC3339)))
	}
	return strings.Join(params, " ")
}

func getGoVersion() string {
	var b strings.Builder
	cmd := exec.Command(utils.Go(), "version")
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	s := strings.Split(b.String(), " ")
	return strings.TrimPrefix(s[2], "go")
}

type Module struct {
	Path      string    `json:"Path"`
	Version   string    `json:"Version"`
	Time      time.Time `json:"Time"`
	Indirect  bool      `json:"Indirect"`
	Dir       string    `json:"Dir"`
	GoMod     string    `json:"GoMod"`
	GoVersion string    `json:"GoVersion"`
}

func getRevaVersion(w *workspace, replacements []Replace) string {
	// we assume here that the reva repository is already available
	// in the current go mod
	if path, ok := isRevaLocalReplacement(replacements); ok {
		var b strings.Builder
		cmd := exec.Command("git", "describe", "--always")
		cmd.Dir = path
		cmd.Stdout = &b
		if err := cmd.Run(); err != nil {
			panic(err)
		}
		return strings.TrimSpace(b.String())
	}

	var b strings.Builder
	cmd := w.newGoCommand(context.Background(), nil, "list", "-m", "-json", revaRepository)
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	var m Module
	if err := json.NewDecoder(strings.NewReader(b.String())).Decode(&m); err != nil {
		panic(err)
	}

	return m.Version
}

type GithubTagRef struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	Object struct {
		Sha  string `json:"sha"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"object"`
}

func getGitCommit(version string, replacements []Replace) string {
	if path, ok := isRevaLocalReplacement(replacements); ok {
		// TODO (gdelmont): is the repository is dirty, this is not actually true
		// we can mark the git commit as "dirty", like "4bbe83eec (*dirty*)"
		var b strings.Builder
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
		cmd.Dir = path
		cmd.Stdout = &b
		if err := cmd.Run(); err != nil {
			panic(err)
		}
		return strings.TrimSpace(b.String())
	}

	// this information is not in the cached go module
	// we need to retrieve this information from github
	url := "https://api.github.com/repos/cs3org/reva/git/refs/tags/" + version
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		panic("status code is not 200")
	}

	var tag GithubTagRef
	if err := json.NewDecoder(res.Body).Decode(&tag); err != nil {
		panic(err)
	}
	return tag.Object.Sha[:9]
}

func getBuildDate() time.Time {
	return time.Now()
}

func (w *workspace) generateBuildFlags(replacements []Replace) buildFlags {
	version := getRevaVersion(w, replacements)
	return buildFlags{
		GitCommit: getGitCommit(version, replacements),
		Version:   version,
		GoVersion: getGoVersion(),
		BuildDate: getBuildDate(),
	}
}
