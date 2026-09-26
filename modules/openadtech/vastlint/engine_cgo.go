//go:build cgo

package vastlint

import govastlint "github.com/aleksUIX/vastlint-go"

func newValidator() (func(string) ([]finding, error), error) {
	return func(xml string) ([]finding, error) {
		result, err := govastlint.Validate(xml)
		if err != nil {
			return nil, err
		}
		out := make([]finding, 0, len(result.Issues))
		for _, issue := range result.Issues {
			out = append(out, finding{ID: issue.ID})
		}
		return out, nil
	}, nil
}
