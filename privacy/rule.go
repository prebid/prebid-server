package privacy

type Rule interface {
	Evaluate(target Component) ActivityResult
}
