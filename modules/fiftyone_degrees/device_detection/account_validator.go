package device_detection

// AccountValidator is a struct that contains an AccountInfoExtractor
// and is used to validate if an account is whitelisted
type AccountValidator struct {
	AccountExtractor AccountInfoExtractor
}

func NewAccountValidator() *AccountValidator {
	return &AccountValidator{
		AccountExtractor: *NewAccountInfoExtractor(),
	}
}

func (x *AccountValidator) IsWhiteListed(cfg Config, req []byte) bool {
	res := false
	accountInfo := x.AccountExtractor.Extract(req)
	if cfg.AccountFilter.AllowList == nil || len(cfg.AccountFilter.AllowList) == 0 {
		res = true
	}

	if accountInfo != nil && cfg.AccountFilter.AllowList != nil && Contains(
		cfg.AccountFilter.AllowList,
		accountInfo.Id,
	) == true {
		res = true
	}

	return res
}
