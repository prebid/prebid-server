package rules

type ResultFunction[T1 any, T2 any] interface {
	Call(payloadIn *T1, payloadOut *T2, meta ResultFunctionMeta) error
	Name() string
}

type ResultFunctionMeta struct {
	SchemaFunctionResults []SchemaFunctionStep
	AnalyticsKey          string
	RuleFired             string
	ModelVersion          string
}

type SchemaFunctionStep struct {
	FuncName   string
	FuncResult string
}
