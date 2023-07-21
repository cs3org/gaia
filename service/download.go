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

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/cs3org/gaia/pkg/builder"
	"github.com/cs3org/reva/pkg/appctx"
)

type downloadRequest struct {
	OS          string
	Arch        string
	RevaVersion string
	Plugins     []string
}

func (s *Builder) download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	req, err := parseDownloadRequest(r)
	if err != nil {
		log.Info().Err(err).Msg("error parsing download request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	plugins, err := parsePlugins(req.Plugins)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info().
		Str("os", req.OS).
		Str("arch", req.Arch).
		Str("reva_version", req.RevaVersion).
		Interface("plugins", plugins).
		Msg("request build of reva")

	b := builder.Builder{
		Platform: builder.Platform{
			OS:   req.OS,
			Arch: req.Arch,
		},
		RevaVersion: req.RevaVersion,
		Log:         log,
		Plugins:     plugins,
		TempFolder:  s.c.BuildFolder,
	}

	output, err := os.CreateTemp(s.c.BinaryTempFolder, "revad")
	if err != nil {
		log.Error().Err(err).Msg("error creating temp file for revad")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer output.Close()
	defer os.RemoveAll(output.Name())

	buildCtx, cancel := context.WithTimeout(ctx, s.c.BuildTimeout)
	defer cancel()
	if err := b.Build(buildCtx, output.Name()); err != nil {
		log.Error().Err(err).Msg("error building reva")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendBinary(ctx, w, binaryRevadName(req.OS, req.Arch, req.RevaVersion), output.Name())
}

func sendBinary(ctx context.Context, w http.ResponseWriter, name, binary string) {
	log := appctx.GetLogger(ctx)

	file, err := os.Open(binary)
	if err != nil {
		log.Error().Err(err).Msgf("error opening file %s", file.Name())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Error().Err(err).Msgf("error statting file %s", file.Name())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	w.Header().Set("Content-Transfer-Encoding", "binary")
	if _, err := io.Copy(w, file); err != nil {
		log.Error().Err(err).Msgf("error sending back reva binary")
		return
	}
}

func binaryRevadName(os, arch, version string) string {
	name := "revad_" + os + "_" + arch
	if version != "" && version != "latest" {
		name += "_" + version
	}
	return name
}

func parseDownloadRequest(r *http.Request) (*downloadRequest, error) {
	var req downloadRequest
	q := r.URL.Query()
	req.OS = q.Get("os")
	if req.OS == "" {
		return nil, errors.New("os is required")
	}
	req.Arch = q.Get("arch")
	if req.Arch == "" {
		return nil, errors.New("arch is required")
	}
	req.RevaVersion = q.Get("version")
	// this can be emtpy, in this case the builder
	// will only build reva without plugins
	req.Plugins = q["plugin"]
	return &req, nil
}

func parsePlugins(plugins []string) ([]builder.Plugin, error) {
	p := make([]builder.Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		p = append(p, parsePlugin(plugin))
	}
	return p, nil
}

func parsePlugin(s string) builder.Plugin {
	var p builder.Plugin
	split := strings.SplitN(s, "@", 2)
	p.RepositoryPath = split[0]
	if len(split) > 1 {
		p.Version = split[1]
	}
	return p
}
