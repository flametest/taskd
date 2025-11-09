package container

import (
	"github.com/flametest/taskd/internal/config"
	"github.com/flametest/taskd/internal/infra/repository"
	"github.com/flametest/vita/vgorm"
)

type Container interface {
	GetRepository() repository.Repository
}

type containerImpl struct {
	repository repository.Repository
}

func NewContainer(config *config.Config) (Container, error) {
	db, err := vgorm.NewDB(config.Datasource)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(db)
	return &containerImpl{repository: repo}, nil
}

func (c *containerImpl) GetRepository() repository.Repository {
	return c.repository
}
