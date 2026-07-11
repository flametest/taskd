package container

import (
	"github.com/flametest/taskd/internal/config"
	"github.com/flametest/taskd/internal/infra/repository"
	"github.com/flametest/vita/vgorm"
	"gorm.io/gorm"
)

type Container interface {
	GetRepository() repository.Repository
	// GetDB returns the underlying *gorm.DB, e.g. for readiness pings.
	GetDB() *gorm.DB
}

type containerImpl struct {
	repository repository.Repository
	db         *gorm.DB
}

func NewContainer(config *config.Config) (Container, error) {
	db, err := vgorm.NewDB(config.Datasource)
	if err != nil {
		return nil, err
	}
	return &containerImpl{
		repository: repository.NewRepository(db),
		db:         db,
	}, nil
}

func (c *containerImpl) GetRepository() repository.Repository {
	return c.repository
}

func (c *containerImpl) GetDB() *gorm.DB {
	return c.db
}
