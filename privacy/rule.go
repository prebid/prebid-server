package privacy

type Rule interface {
	Evaluate(target Component, request ActivityRequest) ActivityResult
}
