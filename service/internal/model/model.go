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
	PackageID   int
	ID          string
	Description string
}
