package stored_requests

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_runABSelectionDistribution(t *testing.T) {
	repeatCount := 100000
	rules := map[string]float64{
		"main":   80,
		"test5":  5.0,
		"test15": 15.0,
	}
	rulesBytes, _ := json.Marshal(rules)
	counts := map[string]int{}
	allowedDeviation := 0.02
	for i := 0; i < repeatCount; i++ {
		gotSelected, err := runABSelection(rulesBytes)
		counts[gotSelected]++
		assert.NoError(t, err)
	}
	for key, val := range rules {
		got := counts[key]
		expected := int(val * float64(repeatCount) / 100)
		deviation := math.Abs(1 - float64(got)/float64(expected))
		assert.True(t, deviation <= allowedDeviation,
			fmt.Sprintf("Case %s: Got %d, wanted %d, deviation %.2f%% > expected %.2f%%", key, got, expected, deviation*100, allowedDeviation*100))
	}
}
