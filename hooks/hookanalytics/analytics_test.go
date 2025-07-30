package hookanalytics

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestAnalytics(t *testing.T) {
	expectedAnalytics := []byte(`
{
  "activities": [
    {
      "name": "device-id",
      "status": "success",
      "results": [
        {
          "status": "success-allow",
          "values": {
            "foo": "bar"
          },
          "appliedto": {
            "impids": [
              "impId1"
            ],
            "request": true
          }
        }
      ]
    },
    {
      "name": "define-blocks",
      "status": "error"
    }
  ]
}
`)

	result := Result{Status: ResultStatusAllow, Values: map[string]interface{}{"foo": "bar"}}
	result.AppliedTo = AppliedTo{ImpIds: []string{"impId1"}, Request: true}

	activity := Activity{Name: "device-id", Status: ActivityStatusSuccess}
	activity.Results = append(activity.Results, result)

	analytics := Analytics{}
	analytics.Activities = append(
		analytics.Activities,
		activity,
		Activity{Name: "define-blocks", Status: ActivityStatusError},
	)

	gotAnalytics, err := jsonutil.Marshal(analytics)
	assert.NoError(t, err, "Failed to marshal analytics: %s", err)
	assert.JSONEq(t, string(expectedAnalytics), string(gotAnalytics))
}
