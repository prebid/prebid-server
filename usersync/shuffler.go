package usersync

import "math/rand"

type shuffler interface {
	shuffle(v []string)
}

type randomShuffler struct{}

func (randomShuffler) shuffle(v []string) {
	rand.Shuffle(len(v), func(i, j int) { v[i], v[j] = v[j], v[i] })
}
