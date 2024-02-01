package uuidutil

import (
	"github.com/gofrs/uuid"
)

type UUIDGenerator interface {
	Generate() (string, error)
}

type UUIDRandomGenerator struct{}

func (UUIDRandomGenerator) Generate() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
