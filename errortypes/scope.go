package errortypes

type Scope int

const (
	ScopeAny Scope = iota
	ScopeDebug
)

type Scoped interface {
	Scope() Scope
}

func ReadScope(err error) Scope {
	if e, ok := err.(Scoped); ok {
		return e.Scope()
	}
	return ScopeAny
}
