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
	"strings"
	"testing"
)

var testPlugins = []Plugin{
	{RepositoryPath: "github.com/cernbox/reva-plugins/cli"},
	{RepositoryPath: revaRepository},
	{RepositoryPath: "example.com/other", Version: "v1.0.0"},
}

func TestWriteMain(t *testing.T) {
	for _, tc := range []struct {
		name   string
		binary Binary
		brand  bool
		want   string
	}{
		{"revad", Revad, false, `package main

import (
	_ "example.com/other"
	_ "github.com/cernbox/reva-plugins/cli"
	revadcmd "github.com/cs3org/reva/v3/cmd/revad"
)

func main() {
	revadcmd.Main()
}
`},
		{"reva", Reva, false, `package main

import (
	_ "example.com/other"
	_ "github.com/cernbox/reva-plugins/cli"
	revacmd "github.com/cs3org/reva/v3/cmd/reva/app"
)

func main() {
	revacmd.Main()
}
`},
		{"reva with a brand", Reva, true, `package main

import (
	_ "embed"

	_ "example.com/other"
	_ "github.com/cernbox/reva-plugins/cli"
	revacmd "github.com/cs3org/reva/v3/cmd/reva/app"
	"github.com/cs3org/reva/v3/pkg/cli/brand"
)

// embedded is the brand of the program (gaia build --brand).
//
//go:embed brand.toml
var embedded []byte

func main() {
	revacmd.Main(revacmd.WithBrand(brand.MustParse(embedded)))
}
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			if err := writeMain(&b, tc.binary, testPlugins, tc.brand); err != nil {
				t.Fatal(err)
			}
			if b.String() != tc.want {
				t.Errorf("got\n%s\nwant\n%s", b.String(), tc.want)
			}
		})
	}
}

func TestWriteMainKeepsThePlugins(t *testing.T) {
	plugins := append([]Plugin(nil), testPlugins...)
	if err := writeMain(new(strings.Builder), Revad, plugins, false); err != nil {
		t.Fatal(err)
	}
	for i := range plugins {
		if plugins[i] != testPlugins[i] {
			t.Errorf("plugin %d changed: %v, was %v", i, plugins[i], testPlugins[i])
		}
	}
}

func TestWriteMainRefusesABrandForRevad(t *testing.T) {
	if err := writeMain(new(strings.Builder), Revad, nil, true); err == nil {
		t.Error("no error")
	}
}

func TestParseBinary(t *testing.T) {
	for in, want := range map[string]Binary{"": Revad, "revad": Revad, "reva": Reva} {
		if got, err := ParseBinary(in); err != nil || got != want {
			t.Errorf("ParseBinary(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseBinary("cernbox"); err == nil {
		t.Error("ParseBinary(cernbox): no error")
	}
}
