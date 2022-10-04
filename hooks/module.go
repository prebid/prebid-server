package hooks

type Module interface {
	Name() string
	Hooks() []interface{}
}
