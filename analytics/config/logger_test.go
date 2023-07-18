package config

import (
	"context"
	"errors"
	"testing"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/stretchr/testify/mock"
)

func TestLogNoModules(t *testing.T) {
	tests := []struct {
		name    string
		modules EnabledAnalytics
	}{
		{
			name:    "modules_are_nil",
			modules: nil,
		},
		{
			name:    "modules_are_empty",
			modules: EnabledAnalytics{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eml := EnabledModuleLogger{
				modules:       tt.modules,
				privacyPolicy: &fakePrivacyPolicy{},
				ctx:           context.Background(),
			}
			ao := analytics.AuctionObject{}
			eml.LogAuctionObject(&ao)
			vo := analytics.VideoObject{}
			eml.LogVideoObject(&vo)
			cso := analytics.CookieSyncObject{}
			eml.LogCookieSyncObject(&cso)
			so := analytics.SetUIDObject{}
			eml.LogSetUIDObject(&so)
			ampo := analytics.AmpObject{}
			eml.LogAmpObject(&ampo)
			ne := analytics.NotificationEvent{}
			eml.LogNotificationEventObject(&ne)
		})
	}
}

func TestLogAuctionObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_by_privacy_policy",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_by_privacy_policy",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_and_denied_by_privacy_policies",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogAuctionObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogAuctionObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			ao := analytics.AuctionObject{}
			eml.LogAuctionObject(&ao)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogAuctionObject", &ao)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogAuctionObject", &ao)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogAuctionObject", &ao)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogAuctionObject", &ao)
			}
		})
	}
}

func TestLogVideoObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_for_module_a_only",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogVideoObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogVideoObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			vo := analytics.VideoObject{}
			eml.LogVideoObject(&vo)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogVideoObject", &vo)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogVideoObject", &vo)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogVideoObject", &vo)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogVideoObject", &vo)
			}
		})
	}
}

func TestLogCookieSyncObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_for_module_a_only",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogCookieSyncObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogCookieSyncObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			cso := analytics.CookieSyncObject{}
			eml.LogCookieSyncObject(&cso)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogCookieSyncObject", &cso)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogCookieSyncObject", &cso)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogCookieSyncObject", &cso)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogCookieSyncObject", &cso)
			}
		})
	}
}

func TestLogSetUIDObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_for_module_a_only",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogSetUIDObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogSetUIDObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			so := analytics.SetUIDObject{}
			eml.LogSetUIDObject(&so)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogSetUIDObject", &so)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogSetUIDObject", &so)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogSetUIDObject", &so)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogSetUIDObject", &so)
			}
		})
	}
}

func TestLogAmpObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_for_module_a_only",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogAmpObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogAmpObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			ao := analytics.AmpObject{}
			eml.LogAmpObject(&ao)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogAmpObject", &ao)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogAmpObject", &ao)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogAmpObject", &ao)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogAmpObject", &ao)
			}
		})
	}
}

func TestLogNotificationEventObject(t *testing.T) {
	tests := []struct {
		name                string
		privacyPolicy       gdpr.PrivacyPolicy
		expectModuleALogged bool
		expectModuleBLogged bool
	}{
		{
			name: "module_logging_allowed_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
			},
			expectModuleALogged: true,
			expectModuleBLogged: true,
		},
		{
			name: "module_logging_denied_for_all_modules",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{},
			},
		},
		{
			name: "module_logging_allowed_for_module_a_only",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}},
			},
			expectModuleALogged: true,
		},
		{
			name: "privacy_policy_error",
			privacyPolicy: &fakePrivacyPolicy{
				allowedAnalytics: map[string]struct{}{"module_a": {}, "module_b": {}},
				policyError:      errors.New("some error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleAMock := &EnabledAnalyticsModuleMock{name: "module_a"}
			moduleAMock.Mock.On("LogNotificationEventObject", mock.Anything).Return()

			moduleBMock := &EnabledAnalyticsModuleMock{name: "module_b"}
			moduleBMock.Mock.On("LogNotificationEventObject", mock.Anything).Return()

			eml := EnabledModuleLogger{
				modules: EnabledAnalytics{
					moduleAMock,
					moduleBMock,
				},
				privacyPolicy: tt.privacyPolicy,
				ctx:           context.Background(),
			}
			ne := analytics.NotificationEvent{}
			eml.LogNotificationEventObject(&ne)

			if tt.expectModuleALogged {
				moduleAMock.Mock.AssertCalled(t, "LogNotificationEventObject", &ne)
			} else {
				moduleAMock.Mock.AssertNotCalled(t, "LogNotificationEventObject", &ne)
			}
			if tt.expectModuleBLogged {
				moduleBMock.Mock.AssertCalled(t, "LogNotificationEventObject", &ne)
			} else {
				moduleBMock.Mock.AssertNotCalled(t, "LogNotificationEventObject", &ne)
			}
		})
	}
}

type EnabledAnalyticsModuleMock struct {
	mock.Mock
	name     string
	vendorID uint16
}

func (ea *EnabledAnalyticsModuleMock) GetName() string {
	return ea.name
}
func (ea *EnabledAnalyticsModuleMock) GetVendorID() uint16 {
	return ea.vendorID
}
func (ea *EnabledAnalyticsModuleMock) LogAuctionObject(ao *analytics.AuctionObject) {
	ea.Called(ao)
}
func (ea *EnabledAnalyticsModuleMock) LogVideoObject(vo *analytics.VideoObject) {
	ea.Called(vo)
}
func (ea *EnabledAnalyticsModuleMock) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	ea.Called(cso)
}
func (ea *EnabledAnalyticsModuleMock) LogSetUIDObject(so *analytics.SetUIDObject) {
	ea.Called(so)
}
func (ea *EnabledAnalyticsModuleMock) LogAmpObject(ao *analytics.AmpObject) {
	ea.Called(ao)
}
func (ea *EnabledAnalyticsModuleMock) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	ea.Called(ne)
}

type fakePrivacyPolicy struct {
	allowedAnalytics map[string]struct{}
	policyError      error
}

func (fpp *fakePrivacyPolicy) Allow(ctx context.Context, name string, gvlID uint16) (bool, error) {

	_, found := fpp.allowedAnalytics[name]
	return found, fpp.policyError
}
