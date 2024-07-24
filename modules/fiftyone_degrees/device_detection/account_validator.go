package device_detection

import "slices"

// AccountValidator is a struct that contains an AccountInfoExtractor
// and is used to validate if an account is allowed
type AccountValidator struct {
	AccountExtractor AccountInfoExtractor
}

func NewAccountValidator() *AccountValidator {
	return &AccountValidator{
		AccountExtractor: *NewAccountInfoExtractor(),
	}
}

func (x *AccountValidator) IsAllowed(cfg Config, req []byte) bool {
	res := false
	accountInfo := x.AccountExtractor.Extract(req)
	if cfg.AccountFilter.AllowList == nil || len(cfg.AccountFilter.AllowList) == 0 {
		res = true
	}

	if accountInfo != nil && cfg.AccountFilter.AllowList != nil && slices.Contains(
		cfg.AccountFilter.AllowList,
		accountInfo.Id,
	) {
		res = true
	}

	return res
}
