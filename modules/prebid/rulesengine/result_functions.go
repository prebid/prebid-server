package rulesengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/v4/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/rules"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// ProcessedAuctionResultFunc is a type alias for a result function that runs in the processed auction request stage.
type ProcessedAuctionResultFunc = rules.ResultFunction[openrtb_ext.RequestWrapper, ProcessedAuctionHookResult]

const (
	ExcludeBiddersName = "excludeBidders"
	IncludeBiddersName = "includeBidders"
)

// NewProcessedAuctionRequestResultFunction is a factory function that creates a new result function based on the provided name and parameters.
// It returns an error if the function name is not recognized or if there is an issue with the parameters.
// The function name is case insensitive.
// The parameters are expected to be in JSON format and will be unmarshalled into the appropriate struct.
// The function returns a rules.ResultFunction that can be used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
// The function is used to create result functions for the rules engine in the Prebid Server.
func NewProcessedAuctionRequestResultFunction(name string, params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	//TODO: make case insensitive converting to lower case
	switch name {
	case ExcludeBiddersName:
		return NewExcludeBidders(params)
	case IncludeBiddersName:
		return NewIncludeBidders(params)
	default:
		return nil, fmt.Errorf("result function %s was not created", name)
	}
}

// NewExcludeBidders is a factory function that creates a new ExcludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an ExcludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The ExcludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewExcludeBidders(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var excludeBiddersParams config.ResultFuncParams
	if err := jsonutil.Unmarshal(params, &excludeBiddersParams); err != nil {
		return nil, err
	}

	if len(excludeBiddersParams.Bidders) == 0 {
		return nil, errors.New("excludeBidders requires at least one bidder to be specified")
	}
	return &ExcludeBidders{Args: excludeBiddersParams}, nil
}

// ExcludeBidders is a struct that holds parameters for excluding bidders in the rules engine.
type ExcludeBidders struct {
	Args config.ResultFuncParams
}

// Call is a method that applies the changes specified in the ExcludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *ExcludeBidders) Call(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult, meta rules.ResultFunctionMeta) error {
	excludedBidders := make(map[string]struct{})
	for _, bidderName := range eb.Args.Bidders {
		excludedBidders[bidderName] = struct{}{} // Ensure the bidder is included in the allowed bidders
	}

	result.HookResult.ChangeSet.ProcessedAuctionRequest().Bidders().Delete(excludedBidders)

	// Only warn about bidders that are actually present in the request, so the debug output
	// reflects what was really removed rather than the rule's full configured exclusion list.
	removedBidders := filterPresentBidders(req, eb.Args.Bidders)
	if len(removedBidders) > 0 {
		result.HookResult.Warnings = append(
			result.HookResult.Warnings,
			buildExclusionWarning(removedBidders, meta),
		)
	}
	return nil
}

func (eb *ExcludeBidders) Name() string {
	return ExcludeBiddersName
}

// filterPresentBidders returns the subset of the given bidders that appear in at least one
// imp's ext.prebid.bidder map, preserving the input order. It lets exclusion warnings mention
// only the bidders that were really removed from the request instead of the configured list.
func filterPresentBidders(req *openrtb_ext.RequestWrapper, bidders []string) []string {
	if req == nil {
		return nil
	}

	present := make(map[string]struct{})
	for _, impWrapper := range req.GetImp() {
		impExt, err := impWrapper.GetImpExt()
		if err != nil {
			continue
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			continue
		}
		for bidderName := range impPrebid.Bidder {
			present[bidderName] = struct{}{}
		}
	}

	filtered := make([]string, 0, len(bidders))
	for _, bidderName := range bidders {
		if _, ok := present[bidderName]; ok {
			filtered = append(filtered, bidderName)
		}
	}
	return filtered
}

// buildExclusionWarning builds a single human-readable warning line describing why a set of
// bidders was removed from the request by the rules engine. The ruleset label (the configured
// ruleset name, falling back to the model group's analytics key when no name is set) and the
// evaluated conditions (SchemaFunctionResults) are only included when present.
func buildExclusionWarning(bidders []string, meta rules.ResultFunctionMeta) string {
	var sb strings.Builder

	if len(bidders) == 1 {
		sb.WriteString(fmt.Sprintf("Bidder [%s] was removed from the request by the rules engine", bidders[0]))
	} else {
		sb.WriteString(fmt.Sprintf("Bidders [%s] were removed from the request by the rules engine", strings.Join(bidders, ", ")))
	}

	// Prefer the human-readable ruleset name; fall back to the analytics key when no name is set.
	rulesetLabel := meta.RulesetName
	if len(rulesetLabel) == 0 {
		rulesetLabel = meta.AnalyticsKey
	}
	if len(rulesetLabel) > 0 {
		sb.WriteString(fmt.Sprintf(" ruleset %q", rulesetLabel))
	}

	if reason := buildSchemaReason(meta.SchemaFunctionResults); len(reason) > 0 {
		sb.WriteString(": ")
		sb.WriteString(reason)
	}

	return sb.String()
}

// buildSchemaReason renders the evaluated schema conditions into a human-readable reason. It is
// generic across every schema function, so any rule added to the engine gets sensible reasoning
// without per-function special casing. It is shared by the exclude and include warning builders.
func buildSchemaReason(steps []rules.SchemaFunctionStep) string {
	if len(steps) == 0 {
		return ""
	}

	reasons := make([]string, 0, len(steps))
	for _, step := range steps {
		reasons = append(reasons, describeSchemaStep(step))
	}
	return strings.Join(reasons, ", ")
}

// describeSchemaStep turns a single evaluated schema condition into plain text of the form
// "<function> rule evaluated to <result>". It is generic across every schema function, so any rule
// added to the engine reads sensibly without per-function special casing. Boolean results
// (true/false) are rendered unquoted; an empty result (e.g. a value function like deviceCountry
// when the request has no geo) is rendered as "(no value)"; any other result (e.g. a country code
// or channel) is quoted.
func describeSchemaStep(step rules.SchemaFunctionStep) string {
	switch step.FuncResult {
	case "true", "false":
		return fmt.Sprintf("%s rule evaluated to %s", step.FuncName, step.FuncResult)
	case "":
		return fmt.Sprintf("%s rule evaluated to (no value)", step.FuncName)
	default:
		return fmt.Sprintf("%s rule evaluated to %q", step.FuncName, step.FuncResult)
	}
}

// NewIncludeBidders is a factory function that creates a new IncludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an IncludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The IncludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewIncludeBidders(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var includeBiddersParams config.ResultFuncParams
	if err := jsonutil.Unmarshal(params, &includeBiddersParams); err != nil {
		return nil, err
	}
	if len(includeBiddersParams.Bidders) == 0 {
		return nil, errors.New("includeBidders requires at least one bidder to be specified")
	}
	return &IncludeBidders{Args: includeBiddersParams}, nil
}

// IncludeBidders is a struct that holds parameters for including bidders in the rules engine.
type IncludeBidders struct {
	Args config.ResultFuncParams
}

// Call is a method that applies the changes specified in the IncludeBidders instance to the provided ChangeSet by creating a mutation.
func (ib *IncludeBidders) Call(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult, meta rules.ResultFunctionMeta) error {
	for _, bidderName := range ib.Args.Bidders {
		result.AllowedBidders[bidderName] = struct{}{} // Ensure the bidder is included in the allowed bidders
	}
	// Record the ruleset context so that, once every ruleset has run and the final allow-list is
	// known, we can surface a debug warning naming the bidders that were implicitly removed because
	// they were not on any include list.
	result.IncludeContexts = append(result.IncludeContexts, meta)
	return nil
}

func (ib *IncludeBidders) Name() string {
	return IncludeBiddersName
}

// biddersRemovedByInclude returns the bidders present in the request that are not in the final
// allow-list accumulated by the include rules. These are the bidders that will be implicitly
// dropped when the allow-list is applied. The result is sorted for deterministic output.
func biddersRemovedByInclude(req *openrtb_ext.RequestWrapper, allowed map[string]struct{}) []string {
	if req == nil {
		return nil
	}

	removed := make(map[string]struct{})
	for _, impWrapper := range req.GetImp() {
		impExt, err := impWrapper.GetImpExt()
		if err != nil {
			continue
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			continue
		}
		for bidderName := range impPrebid.Bidder {
			if _, ok := allowed[bidderName]; !ok {
				removed[bidderName] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(removed))
	for bidderName := range removed {
		result = append(result, bidderName)
	}
	sort.Strings(result)
	return result
}

// appendInclusionWarnings surfaces a debug warning naming the bidders that an include rule
// implicitly removed from the request (bidders present in the request but absent from every
// include list). It is a no-op when no include rule fired or when nothing was removed.
func appendInclusionWarnings(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult) {
	if result == nil || len(result.IncludeContexts) == 0 {
		return
	}

	removed := biddersRemovedByInclude(req, result.AllowedBidders)
	if len(removed) == 0 {
		return
	}

	result.HookResult.Warnings = append(
		result.HookResult.Warnings,
		buildInclusionWarning(removed, result.IncludeContexts),
	)
}

// buildInclusionWarning builds a single human-readable warning line describing why a set of bidders
// was removed from the request because they were not on any include list. Each include rule that
// constrained the request is described by its ruleset label and evaluated conditions when present.
func buildInclusionWarning(bidders []string, contexts []rules.ResultFunctionMeta) string {
	var sb strings.Builder

	if len(bidders) == 1 {
		sb.WriteString(fmt.Sprintf("Bidder [%s] was removed from the request by the rules engine because it was not in the include list", bidders[0]))
	} else {
		sb.WriteString(fmt.Sprintf("Bidders [%s] were removed from the request by the rules engine because they were not in the include list", strings.Join(bidders, ", ")))
	}

	descriptions := make([]string, 0, len(contexts))
	for _, meta := range contexts {
		if desc := describeIncludeContext(meta); len(desc) > 0 {
			descriptions = append(descriptions, desc)
		}
	}
	if len(descriptions) > 0 {
		sb.WriteString(" applied by ")
		sb.WriteString(strings.Join(descriptions, ", "))
	}

	return sb.String()
}

// describeIncludeContext renders a single include rule's ruleset label (the configured ruleset name,
// falling back to the model group's analytics key) and evaluated conditions into text of the form
// `ruleset "X" (deviceCountry rule evaluated to "JPN")`. Either portion is omitted when not present.
func describeIncludeContext(meta rules.ResultFunctionMeta) string {
	rulesetLabel := meta.RulesetName
	if len(rulesetLabel) == 0 {
		rulesetLabel = meta.AnalyticsKey
	}

	reason := buildSchemaReason(meta.SchemaFunctionResults)

	switch {
	case len(rulesetLabel) > 0 && len(reason) > 0:
		return fmt.Sprintf("ruleset %q (%s)", rulesetLabel, reason)
	case len(rulesetLabel) > 0:
		return fmt.Sprintf("ruleset %q", rulesetLabel)
	case len(reason) > 0:
		return reason
	default:
		return ""
	}
}
