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

package crud

import (
	"context"
	"errors"

	"github.com/cs3org/gaia/service/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type drv struct {
	db *gorm.DB
}

func NewSqlite(file string) (Repository, error) {
	db, err := gorm.Open(sqlite.Open(file), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&model.Package{}, &model.Download{}, &model.Plugin{})
	return &drv{db: db}, nil
}

func (d *drv) StorePackage(ctx context.Context, pkg *model.Package) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkg).Error; err != nil {
			return err
		}
		return tx.Create(&model.Download{PackageModule: pkg.Module}).Error
	})
}

func (d *drv) GetPackage(ctx context.Context, module string) (*model.Package, error) {
	pkg := model.Package{Module: module}
	err := d.db.First(&pkg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (d *drv) ListPackages(ctx context.Context) ([]*model.Package, error) {
	var pkgs []*model.Package
	err := d.db.Model(&model.Package{}).
		Preload("Downloads").
		Preload("Plugins").
		Find(&pkgs).Error
	return pkgs, err
}

func (d *drv) IncrementDownloadCounter(ctx context.Context, module string) error {
	return d.db.Model(&model.Download{}).
		Where("package_module = ?", module).
		UpdateColumn("counter", gorm.Expr("counter + ?", 1)).Error
}
