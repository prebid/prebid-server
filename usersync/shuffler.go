package usersync

import "math/rand"

// shuffler changes the order of elements in the slice.
type shuffler interface {
	shuffle(v []string)
}

// randomShuffler randomly changes the order of elements in the slice.
type randomShuffler struct{}

func (randomShuffler) shuffle(v []string) {
	rand.Shuffle(len(v), func(i, j int) { v[i], v[j] = v[j], v[i] })
}
