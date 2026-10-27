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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cs3org/gaia/internal/utils"
)

// The build tests need a reva checkout:
// GAIA_TEST_REVA=/path/to/reva go test ./pkg/builder -run Build
func revaCheckout(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("GAIA_TEST_REVA")
	if dir == "" {
		t.Skip("GAIA_TEST_REVA is not set")
	}
	return dir
}

func revaBuilder(t *testing.T, binary Binary, brand string) *Builder {
	dir := revaCheckout(t)
	b := &Builder{
		Binary:      binary,
		Brand:       brand,
		Plugins:     []Plugin{{RepositoryPath: revaRepository}},
		Replacement: []Replace{{From: revaRepository, To: dir}},
		TempFolder:  t.TempDir(),
	}
	t.Cleanup(b.Close)
	return b
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	home := t.TempDir()
	cmd.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

func TestBuildRevaWithBrand(t *testing.T) {
	b := revaBuilder(t, Reva, "testdata/brand.toml")
	if err := b.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "gaiatest")
	if err := b.Build(t.Context(), bin); err != nil {
		t.Fatal(err)
	}

	if out := run(t, bin, "--help"); !strings.Contains(out, "A test build of the reva CLI") || !strings.Contains(out, "gaiatest [command]") {
		t.Errorf("help does not show the brand:\n%s", out)
	}
	if out := run(t, bin, "version", "--plugins"); !strings.Contains(out, "brand=gaiatest") || strings.Contains(out, "version= ") || strings.Contains(out, "commit= ") {
		t.Errorf("version does not show the version or the brand:\n%s", out)
	}
	run(t, bin, brandCheckArg)

	// the release pipeline cross-compiles the prepared workspace
	for _, goos := range []string{"darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			cmd := exec.Command(utils.Go(), "build", "-o", filepath.Join(t.TempDir(), "gaiatest"), ".")
			cmd.Dir = b.w.folder
			cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}

func TestBuildRevaWithABadBrand(t *testing.T) {
	b := revaBuilder(t, Reva, "testdata/bad-brand.toml")
	err := b.Prepare(t.Context())
	if err == nil || !strings.Contains(err.Error(), "unknown key program.nickname") {
		t.Errorf("got error %v, want unknown key program.nickname", err)
	}
}

func TestBuildRevad(t *testing.T) {
	b := revaBuilder(t, Revad, "")
	if err := b.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "revad")
	if err := b.Build(t.Context(), bin); err != nil {
		t.Fatal(err)
	}
	if out := run(t, bin, "-version"); strings.Contains(out, "version= ") || strings.Contains(out, "commit= ") {
		t.Errorf("revad has no version:\n%s", out)
	}
}

func TestPrepareRefusesABrandForRevad(t *testing.T) {
	b := &Builder{Binary: Revad, Brand: "testdata/brand.toml", TempFolder: t.TempDir()}
	t.Cleanup(b.Close)
	if err := b.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "a brand is for the reva binary") {
		t.Errorf("got error %v, want a brand is for the reva binary", err)
	}
}
