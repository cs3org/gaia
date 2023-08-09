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

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type workspace struct {
	gopath string
	tmp    string
}

func (w *workspace) init() error {
	var err error
	w.gopath, err = os.MkdirTemp("", "gaia-*")
	if err != nil {
		return err
	}
	w.tmp, err = os.MkdirTemp("", "gaia-*")
	if err != nil {
		return err
	}
	return nil
}

func (w *workspace) run(ctx context.Context, name string, args ...string) (string, error) {
	var stdout, stderr strings.Builder

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = w.tmp
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"GOPATH=" + w.gopath}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (w *workspace) downloadModule(ctx context.Context, module string) (string, error) {
	_, err := w.run(ctx, "go", "mod", "init", "tmp")
	if err != nil {
		return "", err
	}
	_, err = w.run(ctx, "go", "get", module)
	if err != nil {
		return "", err
	}

	// go list -m -json github.com/cs3org/gaia/service
	mod, err := w.run(ctx, "go", "list", "-m", "-json", module)
	if err != nil {
		return "", err
	}
	var m struct {
		Dir string `json:"dir"`
	}
	if err := json.Unmarshal([]byte(mod), &m); err != nil {
		return "", err
	}
	return m.Dir, nil
}

func (w *workspace) readManifest(path string) (*Manifest, error) {
	f, err := os.Open(filepath.Join(path, manifest))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m Manifest
	err = json.NewDecoder(f).Decode(&m)
	return &m, err
}

func (w *workspace) close() {
	if w.gopath != "" {
		os.RemoveAll(w.gopath)
	}
	if w.tmp != "" {
		os.RemoveAll(w.tmp)
	}
}
