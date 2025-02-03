package devicedetection

import "slices"

// defaultAccountValidator is a struct that contains an accountInfoExtractor
// and is used to validate if an account is allowed
type defaultAccountValidator struct {
	AccountExtractor accountInfoExtractor
}

func newAccountValidator() *defaultAccountValidator {
	return &defaultAccountValidator{
		AccountExtractor: newAccountInfoExtractor(),
	}
}

func (x defaultAccountValidator) isAllowed(cfg config, req []byte) bool {
	if len(cfg.AccountFilter.AllowList) == 0 {
		return true
	}

	accountInfo := x.AccountExtractor.extract(req)
	if accountInfo != nil && slices.Contains(cfg.AccountFilter.AllowList, accountInfo.Id) {
		return true
	}

	return false
}
