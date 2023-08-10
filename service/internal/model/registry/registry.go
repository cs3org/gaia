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
	"errors"

	"github.com/cs3org/gaia/service/internal/crud"
	"github.com/cs3org/gaia/service/internal/model"
	"github.com/rs/zerolog"
)

// Registry is a place where all the reva plugins
// are registered.
//
// A developer can develop a plugin for reva, and
// register it in the registry.
// A user can browse all the registered plugins.
// Related plugins, in the same module, are grouped
// in packages. A package must contain a list of plugins
// with different IDs.
//
// A package for being included in the registry, must
// store in the root directory of the module, a special
// file named `reva.json`.
// This file contains all the details of the package,
// including the following information:
//   - "author":   name of the author (required)
//   - "licence":  licence of the package (required)
//   - "module":   go module where package is located (required)
//   - "homepage": main page of the package
//   - "doc":      url of the documentation of the package (required)
//   - "plugins":  a list of {id, description} listing and
//     describing the plugins included
//     in the package (required)
type Registry struct {
	repo crud.Repository
}

const manifest = "reva.json"

type Plugin struct {
	ID          string
	Description string
}

type Manifest struct {
	Author   string
	Licence  string
	Module   string
	Homepage string
	Doc      string
	Plugins  []Plugin
}

func (m *Manifest) Valid() bool {
	if m.Author == "" {
		return false
	}
	if m.Licence == "" {
		return false
	}
	if m.Module == "" {
		return false
	}

	// TODO: check doc
	if len(m.Plugins) == 0 {
		return false
	}
	return true
}

// New creates an instance of a Registry.
func New(repository crud.Repository) *Registry {
	return &Registry{repo: repository}
}

// RegisterPackage registers a package, containing a list of related plugins,
// in the registry.
// It checks that the module contains the reva.json file, with all the
// required information, and eventually adds the package info in the registry.
func (r *Registry) RegisterPackage(ctx context.Context, module string) error {
	log := zerolog.Ctx(ctx).With().Str("module", module).Logger()

	var w workspace
	if err := w.init(); err != nil {
		return err
	}
	defer w.close()

	log.Debug().Msg("downloading module")
	path, err := w.downloadModule(ctx, module)
	if err != nil {
		return err
	}

	log.Debug().Msg("reading manifest")
	manifest, err := w.readManifest(path)
	if err != nil {
		return err
	}

	if !manifest.Valid() {
		log.Info().Interface("manifest", manifest).Msg("manifest is not valid")
		return errors.New("manifest is not valid")
	}

	log.Debug().Interface("manifest", manifest).Msg("got manifest")

	plugins := make([]model.Plugin, 0, len(manifest.Plugins))
	for _, p := range manifest.Plugins {
		plugins = append(plugins, model.Plugin{ID: p.ID, Description: p.Description})
	}
	if err := r.repo.StorePackage(ctx, &model.Package{
		Author:   manifest.Author,
		Module:   module,
		Homepage: manifest.Homepage,
		Plugins:  plugins,
	}); err != nil {
		return err
	}

	log.Info().Msg("module registered")
	return nil
}

// ListPackages returns the list of all the packages registered in the registry.
func (r *Registry) ListPackages(ctx context.Context) ([]*model.Package, error) {
	return r.repo.ListPackages(ctx)
}

// IncrementDownloadCounter increments the download counter for the given module.
func (r *Registry) IncrementDownloadCounter(ctx context.Context, module string) error {
	return r.repo.IncrementDownloadCounter(ctx, module)
}

// UpdatePackages is run internally to update the info of all the packages.
// A developer can update the list of plugins in a package, add or remove
// plugins, and this periodical procedure will reflect those changes.
func (r *Registry) UpdatePackages(ctx context.Context) error {
	return errors.New("not yet implemented")
}
