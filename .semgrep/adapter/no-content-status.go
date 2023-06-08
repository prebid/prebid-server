/*
	no-content-status tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func foo() {
	// ruleid: no-content-status-check
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
}

func foo() {
	// ok: no-content-status-check
	if err := adapters.IsResponseStatusCodeNoContent(response); err != nil {
		return nil, nil
	}
}

func foo() {
	// ok: no-content-status-check
	err := adapters.IsResponseStatusCodeNoContent(response)
	if err != nil {
		return nil, nil
	}
}