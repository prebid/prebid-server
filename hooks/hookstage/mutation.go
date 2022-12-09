package hookstage

type MutationType int

const (
	MutationAdd MutationType = iota
	MutationUpdate
	MutationDelete
)

func (mt MutationType) String() string {
	if v, ok := map[MutationType]string{
		MutationAdd:    "add",
		MutationUpdate: "update",
		MutationDelete: "delete",
	}[mt]; ok {
		return v
	}

	return "unknown"
}

type MutationFunc[T any] func(T) (T, error)

type Mutation[T any] struct {
	mutType MutationType
	key     []string        // key indicates path to the modified field
	fn      MutationFunc[T] // fn actual function that makes changes to payload
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

func (c *ChangeSet[T]) AddMutation(fn MutationFunc[T], t MutationType, k ...string) *ChangeSet[T] {
	c.muts = append(c.muts, Mutation[T]{fn: fn, mutType: t, key: k})
	return c
}
