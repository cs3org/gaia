package model

import (
	"context"
	"errors"

	"github.com/cs3org/gaia/service/internal/crud"
	"github.com/cs3org/gaia/service/internal/model"
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

// New creates an instance of a Registry.
func New(repository crud.Repository) *Registry {
	return &Registry{repo: repository}
}

// RegisterPackage registers a package, containing a list of related plugins,
// in the registry.
// It checks that the module contains the reva.json file, with all the
// required information, and eventually adds the package info in the registry.
func (r *Registry) RegisterPackage(ctx context.Context, module string) error {
	// 1. get reva.json (can we use go get <module>?)
	// 2. parse it and get relevant information
	// 3. store in the db
	return errors.New("not yet implemented")
}

// ListPackages returns the list of all the packages registered in the registry.
func (r *Registry) ListPackages(ctx context.Context) ([]*model.Package, error) {
	return nil, errors.New("not yet implemented")
}

// UpdatePackages is run internally to update the info of all the packages.
// A developer can update the list of plugins in a package, add or remove
// plugins, and this periodical procedure will reflect those changes.
func (r *Registry) UpdatePackages(ctx context.Context) error {
	return errors.New("not yet implemented")
}
