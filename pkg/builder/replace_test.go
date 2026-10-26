// Copyright 2018-2026 CERN
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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newModule writes a module example.com/plugins with the package cli/otg.
func newModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cli", "otg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plugins\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestResolveReplace(t *testing.T) {
	root := newModule(t)
	for _, tc := range []struct {
		name string
		in   Replace
		want Replace
	}{
		{"module at its root", Replace{From: "example.com/plugins", To: root}, Replace{From: "example.com/plugins", To: root}},
		{"package at the module root", Replace{From: "example.com/plugins/cli", To: root}, Replace{From: "example.com/plugins", To: root}},
		{"package at its directory", Replace{From: "example.com/plugins/cli", To: filepath.Join(root, "cli")}, Replace{From: "example.com/plugins", To: root}},
		{"package below the directory", Replace{From: "example.com/plugins/cli/otg", To: filepath.Join(root, "cli")}, Replace{From: "example.com/plugins", To: root}},
		{"module with a version", Replace{From: "example.com/plugins/cli", To: "example.com/fork", ToVersion: "v1.0.0"}, Replace{From: "example.com/plugins/cli", To: "example.com/fork", ToVersion: "v1.0.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveReplace(t.Context(), tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestResolveReplaceRelativePath(t *testing.T) {
	root := newModule(t)
	t.Chdir(filepath.Join(root, "cli"))
	got, err := resolveReplace(t.Context(), Replace{From: "example.com/plugins/cli", To: "."})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Replace{From: "example.com/plugins", To: root}); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestResolveReplaceErrors(t *testing.T) {
	root := newModule(t)
	for _, tc := range []struct {
		name string
		in   Replace
		want string
	}{
		{"another module", Replace{From: "example.com/other", To: root}, "holds example.com/plugins (module example.com/plugins), not example.com/other"},
		{"another package", Replace{From: "example.com/plugins/user", To: filepath.Join(root, "cli")}, "holds example.com/plugins/cli"},
		{"no directory", Replace{From: "example.com/plugins", To: filepath.Join(root, "missing")}, "no such file"},
		{"no go.mod", Replace{From: "example.com/plugins", To: t.TempDir()}, "no go.mod"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveReplace(t.Context(), tc.in)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("got error %v, want %q", err, tc.want)
			}
		})
	}
}

func TestResolveReplacements(t *testing.T) {
	root := newModule(t)
	got, err := resolveReplacements(t.Context(), []Replace{
		{From: "example.com/plugins/cli", To: filepath.Join(root, "cli")},
		{From: "example.com/plugins/cli/otg", To: root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Replace{From: "example.com/plugins", To: root}); len(got) != 1 || got[0] != want {
		t.Errorf("got %v, want [%s]", got, want)
	}

	other := newModule(t)
	if _, err := resolveReplacements(t.Context(), []Replace{
		{From: "example.com/plugins", To: root},
		{From: "example.com/plugins/cli", To: other},
	}); err == nil || !strings.Contains(err.Error(), "two replacements for module example.com/plugins") {
		t.Errorf("got error %v, want two replacements", err)
	}
}
