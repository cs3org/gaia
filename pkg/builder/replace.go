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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isDirectoryPath uses the rule of go mod edit: an absolute path, or a path
// that starts with ./ or ../.
func isDirectoryPath(p string) bool {
	if filepath.IsAbs(p) || p == "." || p == ".." {
		return true
	}
	for _, prefix := range []string{"./", "../", `.\`, `..\`} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func findModule(dir string) (string, error) {
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("no go.mod in %s or in its parents", dir)
		}
		d = parent
	}
}

// resolveReplace turns the replacement of a package with a local directory
// into a replacement of the module that holds the directory. A replace for
// the package path would add a second module with the same packages, and the
// build would fail with "ambiguous import". A replacement with a version is
// not changed.
func resolveReplace(ctx context.Context, r Replace) (Replace, error) {
	if r.ToVersion != "" || !isDirectoryPath(r.To) {
		return r, nil
	}
	dir, err := filepath.Abs(r.To)
	if err != nil {
		return Replace{}, err
	}
	if fi, err := os.Stat(dir); err != nil {
		return Replace{}, fmt.Errorf("replacement of %s: %w", r.From, err)
	} else if !fi.IsDir() {
		return Replace{}, fmt.Errorf("replacement of %s: %s is not a directory", r.From, r.To)
	}
	root, err := findModule(dir)
	if err != nil {
		return Replace{}, fmt.Errorf("replacement of %s: %w", r.From, err)
	}
	gomod, err := parseGoModFile(ctx, filepath.Join(root, "go.mod"))
	if err != nil {
		return Replace{}, fmt.Errorf("replacement of %s: %w", r.From, err)
	}
	module := gomod.Module.Path
	if module == "" {
		return Replace{}, fmt.Errorf("replacement of %s: %s has no module path", r.From, filepath.Join(root, "go.mod"))
	}
	pkg := module
	if rel, err := filepath.Rel(root, dir); err == nil && rel != "." {
		pkg += "/" + filepath.ToSlash(rel)
	}
	if r.From != pkg && !strings.HasPrefix(r.From, pkg+"/") {
		return Replace{}, fmt.Errorf("replacement of %s: %s holds %s (module %s), not %s", r.From, r.To, pkg, module, r.From)
	}
	return Replace{From: module, To: root}, nil
}

// resolveReplacements drops the duplicates: two plugins of one local module
// give one replace.
func resolveReplacements(ctx context.Context, rs []Replace) ([]Replace, error) {
	var out []Replace
	seen := map[string]Replace{}
	for _, r := range rs {
		r, err := resolveReplace(ctx, r)
		if err != nil {
			return nil, err
		}
		if prev, ok := seen[r.From]; ok {
			if prev != r {
				return nil, errors.New("two replacements for module " + r.From + ": " + prev.String() + " and " + r.String())
			}
			continue
		}
		seen[r.From] = r
		out = append(out, r)
	}
	return out, nil
}
