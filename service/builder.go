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
	"net/http"
	"os"
	"time"

	"github.com/cs3org/reva"
	"github.com/cs3org/reva/pkg/rhttp/global"
	"github.com/cs3org/reva/pkg/utils/cfg"
	"github.com/go-chi/chi/v5"
)

func init() {
	reva.RegisterPlugin(Builder{})
}

type Builder struct {
	router *chi.Mux
	c      *config
}

type config struct {
	BuildFolder      string        `mapstructure:"build_folder"`
	BinaryTempFolder string        `mapstructure:"binary_temp_folder"`
	BuildTimeout     time.Duration `mapstructure:"build_timeout"`
}

func (c *config) ApplyDefaults() {
	if c.BinaryTempFolder == "" {
		c.BinaryTempFolder, _ = os.MkdirTemp("", "gaia-*")
	}

	if c.BuildTimeout == 0 {
		c.BuildTimeout = 120 * time.Second
	}
}

func (Builder) RevaPlugin() reva.PluginInfo {
	return reva.PluginInfo{
		ID:  "http.services.gaia",
		New: New,
	}
}

func New(ctx context.Context, m map[string]any) (global.Service, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}
	b := Builder{
		router: chi.NewRouter(),
		c:      &c,
	}
	b.initRouter()
	return &b, nil
}

func (s *Builder) initRouter() {
	s.router.Get("/download", s.download)
	s.router.Get("/plugins", s.listPlugins)
	s.router.Post("/plugins", s.registerPlugin)
}

func (s *Builder) listPlugins(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Builder) registerPlugin(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Builder) Handler() http.Handler { return s.router }

func (s *Builder) Prefix() string { return "gaia" }

func (s *Builder) Close() error { return nil }

func (s *Builder) Unprotected() []string { return []string{"/"} }

var _ global.Service = (*Builder)(nil)
var _ global.NewService = New
