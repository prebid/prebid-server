/*
	bad-request-not-ok-status tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func foo() {
	// ruleid: bad-request-not-ok-status-check
	if response.StatusCode == http.StatusBadRequest {
		return nil, nil
	}
}

func bar() {
	// ruleid: bad-request-not-ok-status-check
	if response.StatusCode != http.StatusOK {
		return nil, nil
	}
}

func fooz() {
	// ok: bad-request-not-ok-status-check
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}
}
