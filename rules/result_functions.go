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

func (m *ResultFunctionMeta) appendToSchemaFunctionResults(name string, value string) {
	if len(m.SchemaFunctionResults) == 0 {
		m.SchemaFunctionResults = make([]SchemaFunctionStep, 0, 1)
	}
	m.SchemaFunctionResults = append(m.SchemaFunctionResults, SchemaFunctionStep{
		FuncName:   name,
		FuncResult: value,
	})
	return
}

func (m *ResultFunctionMeta) appendToRuleFired(value string) {
	if len(m.RuleFired) == 0 {
		m.RuleFired = value
	} else {
		m.RuleFired += "|" + value
	}
	return
}

type SchemaFunctionStep struct {
	FuncName   string
	FuncResult string
}
