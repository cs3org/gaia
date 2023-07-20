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

package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

func Go() string {
	g := os.Getenv("GAIA_GO")
	if g != "" {
		return g
	}
	return "go"
}

func FromGoEnv(key ...string) []string {
	if len(key) == 0 {
		return nil
	}

	var b bytes.Buffer
	args := []string{"env", "--json"}
	args = append(args, key...)
	cmd := exec.Command(Go(), args...)
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	var env map[string]string
	if err := json.Unmarshal(b.Bytes(), &env); err != nil {
		panic(err)
	}
	values := make([]string, 0, len(env))
	for k, v := range env {
		values = append(values, k+"="+v)
	}
	return values
}

func KeyFromGoEnv(key string) string {
	values := FromGoEnv(key)
	if len(values) == 0 {
		return ""
	}
	return strings.SplitN(values[0], "=", 2)[1]
}
