package optimizationmodule

import "testing"

// cd to dir
// go test -bench=.

func BenchmarkExecuteRulesRecursive(b *testing.B) {
	rules := BuildRulesTree(GetConf())
	rw := BuildTestRequestWrapper()

	for i := 0; i < b.N; i++ {
		rules.Execute(rw)
	}
}

func BenchmarkExecuteRulesFlat(b *testing.B) {
	rules := BuildRulesTree(GetConf())
	rw := BuildTestRequestWrapper()

	for i := 0; i < b.N; i++ {
		ExecuteFlat(rules, rw)
	}
}
