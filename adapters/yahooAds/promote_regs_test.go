package yahooAds

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func regsExt(j string) *openrtb2.Regs {
	return &openrtb2.Regs{Ext: []byte(j)}
}

// TestPromoteRegsExtTo26 exercises the regs.ext promotion in isolation; the
// JSON fixtures cover the same logic end-to-end through MakeRequests.
func TestPromoteRegsExtTo26(t *testing.T) {
	tests := []struct {
		name              string
		inRegs            *openrtb2.Regs
		wantCoppa         int8
		wantGPP           string
		wantGPPSID        []int8
		wantExt           string // expected regs.ext JSON when it should remain
		wantNoExt         bool   // expect regs.ext to be empty/nil after the call
		wantRegsUnchanged bool
		wantWarning       bool // expect one errortypes.Warning
	}{
		{
			name:              "nil_regs",
			inRegs:            nil,
			wantRegsUnchanged: true,
		},
		{
			name:              "regs_without_ext",
			inRegs:            &openrtb2.Regs{COPPA: 0},
			wantNoExt:         true,
			wantRegsUnchanged: true,
		},
		{
			name:              "regs_with_empty_ext_bytes",
			inRegs:            &openrtb2.Regs{Ext: []byte{}},
			wantNoExt:         true,
			wantRegsUnchanged: true,
		},
		{
			name:              "regs_with_empty_object_ext",
			inRegs:            regsExt(`{}`),
			wantExt:           `{}`,
			wantRegsUnchanged: true,
		},
		{
			name:              "regs_ext_array_not_object",
			inRegs:            regsExt(`[1,2,3]`),
			wantExt:           `[1,2,3]`,
			wantRegsUnchanged: true,
			wantWarning:       true,
		},
		{
			name:              "regs_ext_number_not_object",
			inRegs:            regsExt(`42`),
			wantExt:           `42`,
			wantRegsUnchanged: true,
			wantWarning:       true,
		},
		{
			// COPPA is top-level since OpenRTB 2.5 and is not promoted from ext.
			name:              "coppa_in_ext_not_promoted",
			inRegs:            regsExt(`{"coppa":1}`),
			wantExt:           `{"coppa":1}`,
			wantRegsUnchanged: true,
		},
		{
			name:      "promote_gpp_only",
			inRegs:    regsExt(`{"gpp":"DBA"}`),
			wantGPP:   "DBA",
			wantNoExt: true,
		},
		{
			name:       "promote_gpp_sid_only",
			inRegs:     regsExt(`{"gpp_sid":[6,7]}`),
			wantGPPSID: []int8{6, 7},
			wantNoExt:  true,
		},
		{
			name:       "promote_both_gpp_fields_together",
			inRegs:     regsExt(`{"gpp":"X","gpp_sid":[6]}`),
			wantGPP:    "X",
			wantGPPSID: []int8{6},
			wantNoExt:  true,
		},
		{
			name:       "promote_with_unrelated_siblings_coppa_gpc",
			inRegs:     regsExt(`{"coppa":1,"gpp":"X","gpp_sid":[6],"gpc":"1"}`),
			wantGPP:    "X",
			wantGPPSID: []int8{6},
			wantExt:    `{"coppa":1,"gpc":"1"}`,
		},
		{
			name:    "promote_gpp_keep_dsa_object_in_ext",
			inRegs:  regsExt(`{"gpp":"X","dsa":{"dsarequired":1}}`),
			wantGPP: "X",
			wantExt: `{"dsa":{"dsarequired":1}}`,
		},
		{
			name:       "wrong_type_gpp_number_stays_in_ext",
			inRegs:     regsExt(`{"gpp":123,"gpp_sid":[6]}`),
			wantGPPSID: []int8{6},
			wantExt:    `{"gpp":123}`,
		},
		{
			name:    "wrong_type_gpp_sid_string_stays_in_ext",
			inRegs:  regsExt(`{"gpp":"X","gpp_sid":"oops"}`),
			wantGPP: "X",
			wantExt: `{"gpp_sid":"oops"}`,
		},
		{
			name:              "wrong_type_only_regs_untouched",
			inRegs:            regsExt(`{"gpp":123}`),
			wantExt:           `{"gpp":123}`,
			wantRegsUnchanged: true,
		},
		{
			name:      "promote_gpp_empty_string_consumed",
			inRegs:    regsExt(`{"gpp":""}`),
			wantGPP:   "",
			wantNoExt: true,
		},
		{
			name:       "promote_gpp_sid_empty_array",
			inRegs:     regsExt(`{"gpp_sid":[]}`),
			wantGPPSID: []int8{},
			wantNoExt:  true,
		},
		{
			name: "top_gpp_already_set_ext_duplicate_stripped",
			inRegs: &openrtb2.Regs{
				GPP: "EXISTING",
				Ext: []byte(`{"gpp":"EXT_VALUE"}`),
			},
			wantGPP:   "EXISTING",
			wantNoExt: true,
		},
		{
			name: "top_gpp_sid_already_set_ext_duplicate_stripped",
			inRegs: &openrtb2.Regs{
				GPPSID: []int8{1, 2},
				Ext:    []byte(`{"gpp_sid":[6]}`),
			},
			wantGPPSID: []int8{1, 2},
			wantNoExt:  true,
		},
		{
			name: "top_gpp_set_duplicate_stripped_sibling_kept",
			inRegs: &openrtb2.Regs{
				GPP: "KEEP",
				Ext: []byte(`{"gpp":"stale","gpc":"1"}`),
			},
			wantGPP: "KEEP",
			wantExt: `{"gpc":"1"}`,
		},
		{
			name: "top_gpp_set_wrong_typed_duplicate_also_stripped",
			inRegs: &openrtb2.Regs{
				GPP: "KEEP",
				Ext: []byte(`{"gpp":123}`),
			},
			wantGPP:   "KEEP",
			wantNoExt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warnings := promoteRegsExtTo26(tt.inRegs)

			if tt.wantWarning {
				require.Len(t, warnings, 1)
				assert.IsType(t, &errortypes.Warning{}, warnings[0])
			} else {
				assert.Empty(t, warnings)
			}

			if tt.wantRegsUnchanged {
				assert.Same(t, tt.inRegs, got, "regs pointer should NOT change for no-op input")
			} else {
				assert.NotSame(t, tt.inRegs, got, "regs should be swapped to a fresh copy after promotion")
			}

			if got == nil {
				return
			}

			assert.Equal(t, tt.wantCoppa, got.COPPA, "regs.coppa")
			assert.Equal(t, tt.wantGPP, got.GPP, "regs.gpp")
			assert.Equal(t, tt.wantGPPSID, got.GPPSID, "regs.gpp_sid")

			switch {
			case tt.wantNoExt:
				assert.Empty(t, got.Ext, "regs.ext should be empty/nil")
			case tt.wantExt != "":
				var gotExt, wantExt any
				require.NoError(t, json.Unmarshal(got.Ext, &gotExt), "regs.ext should be valid JSON")
				require.NoError(t, json.Unmarshal([]byte(tt.wantExt), &wantExt))
				assert.Equal(t, wantExt, gotExt, "regs.ext content")
			default:
				t.Fatal("test case must set wantNoExt or wantExt")
			}
		})
	}
}

// The original *Regs must never be mutated: other adapters and per-impression
// copies must keep seeing the publisher's original request shape.
func TestPromoteRegsExtTo26_DoesNotMutateOriginalRegs(t *testing.T) {
	originalExt := []byte(`{"coppa":1,"gpp":"X","gpp_sid":[6],"gpc":"1"}`)
	originalRegs := &openrtb2.Regs{Ext: append([]byte(nil), originalExt...)}

	beforeCoppa := originalRegs.COPPA
	beforeGPP := originalRegs.GPP
	beforeGPPSID := append([]int8(nil), originalRegs.GPPSID...)
	beforeExt := append([]byte(nil), originalRegs.Ext...)

	got, warnings := promoteRegsExtTo26(originalRegs)

	assert.Empty(t, warnings)
	assert.Equal(t, beforeCoppa, originalRegs.COPPA, "original Regs.COPPA was mutated")
	assert.Equal(t, beforeGPP, originalRegs.GPP, "original Regs.GPP was mutated")
	assert.Equal(t, beforeGPPSID, originalRegs.GPPSID, "original Regs.GPPSID was mutated")
	assert.Equal(t, beforeExt, []byte(originalRegs.Ext), "original Regs.Ext bytes were mutated")

	assert.NotSame(t, originalRegs, got, "promoted regs should be a fresh copy")
	assert.Equal(t, int8(0), got.COPPA, "coppa is not promoted from ext")
	assert.Equal(t, "X", got.GPP)
	assert.Equal(t, []int8{6}, got.GPPSID)
	assert.JSONEq(t, `{"coppa":1,"gpc":"1"}`, string(got.Ext), "coppa and gpc stay in ext")
}

// A regs.ext warning must not drop the impression: the bid request still goes
// out, with regs.ext passed through untouched.
func TestMakeRequestsMalformedRegsExtWarning(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooAds, config.Adapter{}, config.Server{})
	require.NoError(t, buildErr)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "test-imp-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
			Ext:    []byte(`{"bidder":{"dcn":"dcn1","pos":"pos2"}}`),
		}},
		Site: &openrtb2.Site{ID: "test-site-id"},
		Regs: &openrtb2.Regs{Ext: []byte(`[1,2,3]`)},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	require.Len(t, errs, 1)
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	require.Len(t, reqs, 1, "impression must not be dropped on a regs.ext warning")
	assert.Contains(t, string(reqs[0].Body), `"ext":[1,2,3]`, "regs.ext should pass through untouched")
}
