package rules

type ResultFunction[T1 any, T2 any] interface {
	Call(payloadIn *T1, payloadOut *T2, funcMeta ResultFuncMetadata) error
	Name() string
}

type ResultFuncMetadata struct {
	SchemaFunctionResults []SchemaFunctionStep
	AnalyticsKey          string
	RuleFired             string
	ModelVersion          string
}

type SchemaFunctionStep struct {
	FuncName   string
	FuncResult string
}
