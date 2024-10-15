package mediasquare

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestParserDSA(t *testing.T) {
	tests := []struct {
		// tests inputs
		parser  parserDSA
		request openrtb2.BidRequest
		// tests expected-results
		getValue   string
		setContent error
		expected   parserDSA
	}{
		{
			parser: parserDSA{},
			request: openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":"dsa-ok"}`),
				},
			},

			getValue:   "dsa-ok",
			setContent: nil,
			expected:   parserDSA{DSA: "dsa-ok"},
		},
		{
			parser: parserDSA{},
			request: openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"no-dsa":"no-dsa"}`),
				},
			},

			getValue:   "",
			setContent: nil,
			expected:   parserDSA{},
		},
		{
			parser: parserDSA{},
			request: openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(``),
				},
			},

			getValue:   "",
			setContent: errorWritter("<setContent(*parserDSA)> extJsonBytes", nil, true),
			expected:   parserDSA{},
		},
	}
	for index, test := range tests {
		assert.Equal(t, test.getValue, test.parser.getValue(&test.request), fmt.Sprintf("getValue >> index: %d", index))
		assert.Equal(t, test.setContent, test.parser.setContent(test.request.Regs.Ext), fmt.Sprintf("setContent >> index: %d", index))
		assert.Equal(t, test.expected, test.parser, fmt.Sprintf("exactValue >> index: %d", index))
	}

	var pError parserDSA
	assert.Equal(t, pError.setContent([]byte(`{invalid json}`)),
		errorWritter("<setContent(*parserDSA)> extJsonBytes", errors.New("invalid character 'i' looking for beginning of object key string"), false))
}

func TestParserGDPR(t *testing.T) {
	tests := []struct {
		//tests inputs
		parser       parserGDPR
		request      openrtb2.BidRequest
		extJsonBytes []byte
		// tests expected-results
		getValue struct {
			field string
			value string
		}
		setContent error
		value      string
	}{
		{
			parser:       parserGDPR{},
			request:      openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"gdpr":"gdpr-ok-user"}`)}},
			extJsonBytes: []byte(`{"gdpr":"gdpr-ok-extjson"}`),

			getValue: struct {
				field string
				value string
			}{field: "consent_string", value: "gdpr-ok-user"},
			setContent: nil,
			value:      "gdpr-ok-extjson",
		},
		{
			parser:       parserGDPR{},
			request:      openrtb2.BidRequest{User: &openrtb2.User{Consent: "consent-ok-user"}},
			extJsonBytes: []byte(`{"gdpr":"gdpr-ok","consent":"consent-ok"}`),

			getValue: struct {
				field string
				value string
			}{field: "consent_string", value: "consent-ok-user"},
			setContent: nil,
			value:      "consent-ok",
		},
		{
			parser:       parserGDPR{},
			request:      openrtb2.BidRequest{},
			extJsonBytes: []byte(""),

			getValue: struct {
				field string
				value string
			}{field: "consent_string", value: ""},
			setContent: errorWritter("<setContent(*parserGDPR)> extJsonBytes", nil, true),
			value:      "",
		},
		{
			parser:       parserGDPR{},
			request:      openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: IntAsPtrInt8(0)}},
			extJsonBytes: []byte(""),

			getValue: struct {
				field string
				value string
			}{field: "consent_requirement", value: "false"},
			setContent: nil,
			value:      "",
		},
		{
			parser:       parserGDPR{},
			request:      openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: IntAsPtrInt8(1)}},
			extJsonBytes: []byte(""),

			getValue: struct {
				field string
				value string
			}{field: "consent_requirement", value: "true"},
			setContent: nil,
			value:      "",
		},
		{
			parser: parserGDPR{},
			getValue: struct {
				field string
				value string
			}{field: "null", value: ""},
		},
	}

	for index, test := range tests {
		switch test.getValue.field {
		case "consent_string":
			assert.Equal(t, test.getValue.value, test.parser.getValue("consent_string", &(test.request)), fmt.Sprintf("[consent_string]: getValue >> index: %d", index))
			assert.Equal(t, test.setContent, test.parser.setContent(test.extJsonBytes), fmt.Sprintf("setContent >> index: %d", index))
			assert.Equal(t, test.value, test.parser.value(), fmt.Sprintf("value >> index: %d", index))
		case "consent_requirement":
			assert.Equal(t, test.getValue.value, test.parser.getValue("consent_requirement", &(test.request)), fmt.Sprintf("[consent_requirement]: getValue >> index: %d", index))
		case "null":
			assert.Equal(t, test.getValue.value, test.parser.getValue("null", nil), fmt.Sprintf("[consent_requirement]: getValue >> index: %d", index))
		}
	}

	var pError parserGDPR
	assert.Equal(t, pError.setContent([]byte(`{invalid json}`)),
		errorWritter("<setContent(*parserGDPR)> extJsonBytes", errors.New("invalid character 'i' looking for beginning of object key string"), false))
}

func IntAsPtrInt8(i int) *int8 {
	val := int8(i)
	return &val
}
