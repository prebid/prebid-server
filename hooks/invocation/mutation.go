package invocation

type MutationType int

const (
	MutationUpdate MutationType = iota
	MutationDelete
)

type mutationFunc[T any] func(T) (T, error)

type Mutation[T any] struct {
	mutType MutationType // mutType used to determine type of changes made by hook
	key     []string     // key indicates path to the modified key
	fn      mutationFunc[T]
}

func (m Mutation[T]) Type() MutationType {
	return m.mutType
}

func (m Mutation[T]) Key() []string {
	return m.key
}

func (m Mutation[T]) Apply(p T) (T, error) {
	return m.fn(p)
}

type ChangeSet[T any] struct {
	muts []Mutation[T]
}

func (c *ChangeSet[T]) Mutations() []Mutation[T] {
	return c.muts
}

func (c *ChangeSet[T]) AddMutation(fn mutationFunc[T], t MutationType, k ...string) *ChangeSet[T] {
	c.muts = append(c.muts, Mutation[T]{fn: fn, mutType: t, key: k})
	return c
}
