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
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/cs3org/gaia/service/internal/crud"
	model "github.com/cs3org/gaia/service/internal/model/registry"
	"github.com/go-chi/chi/v5"
)

type Builder struct {
	router *chi.Mux
	c      *Config
	reg    *model.Registry
}

type Config struct {
	BuildFolder      string        `mapstructure:"build_folder"`
	BinaryTempFolder string        `mapstructure:"binary_temp_folder"`
	BuildTimeout     time.Duration `mapstructure:"build_timeout"`
	DBFile           string        `mapstructure:"db_file"`

	tmpFile bool
}

func (c *Config) ApplyDefaults() {
	if c.BinaryTempFolder == "" {
		c.BinaryTempFolder, _ = os.MkdirTemp("", "gaia-*")
	}

	if c.BuildTimeout == 0 {
		c.BuildTimeout = 120 * time.Second
	}

	if c.DBFile == "" {
		tmp, err := os.CreateTemp("", "*")
		if err != nil {
			panic(err)
		}
		c.DBFile = tmp.Name()
		c.tmpFile = true
	}
}

func New(c *Config) (*Builder, error) {
	c.ApplyDefaults()
	db, err := crud.NewSqlite(c.DBFile)
	if err != nil {
		return nil, err
	}
	registry := model.New(db)
	b := Builder{
		router: chi.NewRouter(),
		c:      c,
		reg:    registry,
	}
	b.initRouter()
	return &b, nil
}

func (s *Builder) initRouter() {
	s.router.Get("/download", s.download)
	s.router.Get("/plugins", s.listPlugins)
	s.router.Post("/plugins", s.registerPlugin)
}

func writeError(err error, code int, w http.ResponseWriter) {
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}

type plugin struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type packageRes struct {
	Module   string   `json:"module"`
	Download int      `json:"download"`
	Listed   bool     `json:"listed"`
	Plugins  []plugin `json:"plugins"`
}

func (s *Builder) listPlugins(w http.ResponseWriter, r *http.Request) {
	list, err := s.reg.ListPackages(r.Context())
	if err != nil {
		writeError(err, http.StatusInternalServerError, w)
		return
	}

	res := make([]packageRes, 0, len(list))
	for _, p := range list {
		plugins := make([]plugin, 0, len(p.Plugins))
		for _, plug := range p.Plugins {
			plugins = append(plugins, plugin{
				ID:          plug.ID,
				Description: plug.Description,
			})
		}
		res = append(res, packageRes{
			Module:   p.Module,
			Download: p.Downloads.Counter,
			Listed:   true,
			Plugins:  plugins,
		})
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		writeError(err, http.StatusInternalServerError, w)
		return
	}
}

type registerPluginRequest struct {
	Module string `json:"module"`
}

func (s *Builder) registerPlugin(w http.ResponseWriter, r *http.Request) {
	var req registerPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(err, http.StatusBadRequest, w)
		return
	}
	if err := s.reg.RegisterPackage(r.Context(), req.Module); err != nil {
		writeError(err, http.StatusBadRequest, w)
		return
	}
}

func (s *Builder) Handler() http.Handler { return s.router }

func (s *Builder) Close() error {
	if s.c.tmpFile {
		return os.RemoveAll(s.c.DBFile)
	}
	return nil
}
