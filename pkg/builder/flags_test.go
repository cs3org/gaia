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
	"time"
)

func TestBuildFlagsFormat(t *testing.T) {
	f := buildFlags{
		GitCommit: "7ecb07ae9",
		Version:   "v3.14.0-103-g7ecb07ae9",
		GoVersion: "1.27.3",
		BuildDate: time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC),
	}
	want := []string{
		"-X github.com/cs3org/reva/v3/pkg/rversion.GitCommit=7ecb07ae9",
		"-X github.com/cs3org/reva/v3/pkg/rversion.Version=v3.14.0-103-g7ecb07ae9",
		"-X github.com/cs3org/reva/v3/pkg/rversion.GoVersion=1.27.3",
		"-X github.com/cs3org/reva/v3/pkg/rversion.BuildDate=2026-10-26T20:00:00Z",
		"-X github.com/cs3org/reva/v3/cmd/revad.gitCommit=7ecb07ae9",
		"-X github.com/cs3org/reva/v3/cmd/revad.version=v3.14.0-103-g7ecb07ae9",
		"-X github.com/cs3org/reva/v3/cmd/revad.goVersion=1.27.3",
		"-X github.com/cs3org/reva/v3/cmd/revad.buildDate=2026-10-26T20:00:00Z",
	}
	if got := f.Format(); got != strings.Join(want, " ") {
		t.Errorf("got\n%s\nwant\n%s", got, strings.Join(want, " "))
	}
	if got := (buildFlags{Version: "v3.14.0"}).Format(); strings.Contains(got, "BuildDate") || strings.Contains(got, "GitCommit") {
		t.Errorf("empty values are set: %s", got)
	}
}
