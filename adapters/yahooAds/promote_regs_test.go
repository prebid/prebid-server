package yahooAds

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		},
		{
			name:              "regs_ext_number_not_object",
			inRegs:            regsExt(`42`),
			wantExt:           `42`,
			wantRegsUnchanged: true,
		},
		{
			name:      "promote_coppa_only",
			inRegs:    regsExt(`{"coppa":1}`),
			wantCoppa: 1,
			wantNoExt: true,
		},
		{
			// Pins the COPPA int8 limitation documented in promoteRegsExtTo26:
			// an explicit top-level coppa:0 is indistinguishable from unset.
			name: "explicit_coppa_zero_overridden_by_ext",
			inRegs: &openrtb2.Regs{
				COPPA: 0,
				Ext:   []byte(`{"coppa":1}`),
			},
			wantCoppa: 1,
			wantNoExt: true,
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
			name:       "promote_all_three_together",
			inRegs:     regsExt(`{"coppa":1,"gpp":"X","gpp_sid":[6]}`),
			wantCoppa:  1,
			wantGPP:    "X",
			wantGPPSID: []int8{6},
			wantNoExt:  true,
		},
		{
			name:       "promote_with_unrelated_sibling_gpc",
			inRegs:     regsExt(`{"coppa":1,"gpp":"X","gpp_sid":[6],"gpc":"1"}`),
			wantCoppa:  1,
			wantGPP:    "X",
			wantGPPSID: []int8{6},
			wantExt:    `{"gpc":"1"}`,
		},
		{
			name:    "promote_gpp_keep_dsa_object_in_ext",
			inRegs:  regsExt(`{"gpp":"X","dsa":{"dsarequired":1}}`),
			wantGPP: "X",
			wantExt: `{"dsa":{"dsarequired":1}}`,
		},
		{
			name:       "wrong_type_gpp_number_stays_in_ext",
			inRegs:     regsExt(`{"coppa":1,"gpp":123,"gpp_sid":[6]}`),
			wantCoppa:  1,
			wantGPPSID: []int8{6},
			wantExt:    `{"gpp":123}`,
		},
		{
			name:    "wrong_type_coppa_string_stays_in_ext",
			inRegs:  regsExt(`{"coppa":"1","gpp":"X"}`),
			wantGPP: "X",
			wantExt: `{"coppa":"1"}`,
		},
		{
			name:      "wrong_type_gpp_sid_string_stays_in_ext",
			inRegs:    regsExt(`{"coppa":1,"gpp_sid":"oops"}`),
			wantCoppa: 1,
			wantExt:   `{"gpp_sid":"oops"}`,
		},
		{
			name:    "wrong_type_coppa_overflow_stays_in_ext",
			inRegs:  regsExt(`{"coppa":300,"gpp":"X"}`),
			wantGPP: "X",
			wantExt: `{"coppa":300}`,
		},
		{
			name: "top_coppa_already_set_ext_duplicate_stripped",
			inRegs: &openrtb2.Regs{
				COPPA: 1,
				Ext:   []byte(`{"coppa":0}`),
			},
			wantCoppa: 1,
			wantNoExt: true,
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
			req := &openrtb2.BidRequest{Regs: tt.inRegs}
			origPtr := req.Regs

			promoteRegsExtTo26(req)

			if tt.wantRegsUnchanged {
				assert.Same(t, origPtr, req.Regs, "request.Regs pointer should NOT change for no-op input")
			} else {
				assert.NotSame(t, origPtr, req.Regs, "request.Regs should be swapped to a fresh copy after promotion")
			}

			if req.Regs == nil {
				return
			}

			assert.Equal(t, tt.wantCoppa, req.Regs.COPPA, "regs.coppa")
			assert.Equal(t, tt.wantGPP, req.Regs.GPP, "regs.gpp")
			assert.Equal(t, tt.wantGPPSID, req.Regs.GPPSID, "regs.gpp_sid")

			switch {
			case tt.wantNoExt:
				assert.Empty(t, req.Regs.Ext, "regs.ext should be empty/nil")
			case tt.wantExt != "":
				var got, want any
				require.NoError(t, json.Unmarshal(req.Regs.Ext, &got), "regs.ext should be valid JSON")
				require.NoError(t, json.Unmarshal([]byte(tt.wantExt), &want))
				assert.Equal(t, want, got, "regs.ext content")
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

	req := &openrtb2.BidRequest{Regs: originalRegs}
	promoteRegsExtTo26(req)

	assert.Equal(t, beforeCoppa, originalRegs.COPPA, "original Regs.COPPA was mutated")
	assert.Equal(t, beforeGPP, originalRegs.GPP, "original Regs.GPP was mutated")
	assert.Equal(t, beforeGPPSID, originalRegs.GPPSID, "original Regs.GPPSID was mutated")
	assert.Equal(t, beforeExt, []byte(originalRegs.Ext), "original Regs.Ext bytes were mutated")

	assert.NotSame(t, originalRegs, req.Regs, "request.Regs should point to a fresh copy")
	assert.Equal(t, int8(1), req.Regs.COPPA)
	assert.Equal(t, "X", req.Regs.GPP)
	assert.Equal(t, []int8{6}, req.Regs.GPPSID)
}
