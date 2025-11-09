package container

import "github.com/flametest/taskd/internal/infra/repository"

type Container interface {
	GetRepository() repository.Repository
}

type containerImpl struct {
	repository repository.Repository
}

func NewContainer(repo repository.Repository) Container {
	return &containerImpl{repository: repo}
}

func (c *containerImpl) GetRepository() repository.Repository {
	return c.repository
}
