package crud

import (
	"context"

	"github.com/cs3org/gaia/service/internal/model"
)

// Repository is an interface for a repository
// storing packages.
type Repository interface {
	StorePackage(ctx context.Context, pkg *model.Package) error
	ListPackages(ctx context.Context) ([]*model.Package, error)
	IncrementDownloadCounter(ctx context.Context, module string) error
}
