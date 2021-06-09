package criteo

import "github.com/gofrs/uuid"

type slotIDGenerator interface {
	NewSlotID() (string, error)
}

type randomSlotIDGenerator struct{}

func newRandomSlotIDGenerator() randomSlotIDGenerator {
	return randomSlotIDGenerator{}
}

func (g randomSlotIDGenerator) NewSlotID() (string, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	return guid.String(), nil
}

type fakeSlotIDGenerator struct {
	fakeSlotID string
}

func newFakeGuidGenerator(fakeSlotID string) fakeSlotIDGenerator {
	return fakeSlotIDGenerator{
		fakeSlotID,
	}
}

func (f fakeSlotIDGenerator) NewSlotID() (string, error) {
	return f.fakeSlotID, nil
}
