package doohqty

import (
	"fmt"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func resolveImpressionLookups(request *openrtb_ext.RequestWrapper, accountID string, lookupPaths []string) (map[int]lookupKey, []lookupKey, []string) {
	assignments := make(map[int]lookupKey)
	uniqueLookups := make([]lookupKey, 0)
	warnings := make([]string, 0)

	if request == nil || request.BidRequest == nil || request.DOOH == nil {
		return assignments, uniqueLookups, warnings
	}

	seen := make(map[lookupKey]struct{})
	for index, imp := range request.GetImp() {
		lookup, ok := resolveImpressionLookup(request, imp, accountID, lookupPaths)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("no DOOH qty lookup key resolved for imp index %d", index))
			continue
		}

		assignments[index] = lookup
		if _, exists := seen[lookup]; !exists {
			seen[lookup] = struct{}{}
			uniqueLookups = append(uniqueLookups, lookup)
		}
	}

	return assignments, uniqueLookups, warnings
}

func resolveImpressionLookup(request *openrtb_ext.RequestWrapper, imp *openrtb_ext.ImpWrapper, accountID string, lookupPaths []string) (lookupKey, bool) {
	for _, path := range lookupPaths {
		value := lookupPathValue(request, imp, path)
		if value == "" {
			continue
		}

		return lookupKey{
			AccountID: accountID,
			Path:      path,
			Key:       value,
		}, true
	}

	return lookupKey{}, false
}

func lookupPathValue(request *openrtb_ext.RequestWrapper, imp *openrtb_ext.ImpWrapper, path string) string {
	if request == nil || request.BidRequest == nil {
		return ""
	}

	switch path {
	case lookupPathDOOHID:
		if request.DOOH != nil {
			return request.DOOH.ID
		}
	case lookupPathDOOHName:
		if request.DOOH != nil {
			return request.DOOH.Name
		}
	case lookupPathDOOHPublisherID:
		if request.DOOH != nil && request.DOOH.Publisher != nil {
			return request.DOOH.Publisher.ID
		}
	case lookupPathImpID:
		if imp != nil && imp.Imp != nil {
			return imp.ID
		}
	case lookupPathImpTagID:
		if imp != nil && imp.Imp != nil {
			return imp.TagID
		}
	}

	return ""
}
