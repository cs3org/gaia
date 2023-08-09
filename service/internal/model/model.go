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

package model

import (
	"time"

	"gorm.io/gorm"
)

// Package holds the information of a package,
// containing a list of plugins for reva.
type Package struct {
	Author    string
	Module    string `gorm:"primaryKey"`
	Homepage  string
	Plugins   []Plugin `gorm:"OnDelete:CASCADE"`
	Downloads Download `gorm:"OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Download store the number of downloads
// of a package.
type Download struct {
	PackageModule string `gorm:"primaryKey,foreignKey:Module"`
	Counter       int
}

// Plugin holds the information of a plugin,
// the id, in the form of <namespace>.<name>,
// and a description, explaining the purpose
// of the plugin.
type Plugin struct {
	PackageModule string `gorm:"primaryKey,foreignKey:Module"`
	ID            string `gorm:"primaryKey"`
	Description   string
}
