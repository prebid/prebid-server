package privacysandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Topic struct {
	SegTax   int    `json:"segtax,omitempty"`
	SegClass string `json:"segclass,omitempty"`
	SegIDs   []int  `json:"segids,omitempty"`
}

// ParseTopicsFromHeader parses the Sec-Browsing-Topics header data into Topics object
func ParseTopicsFromHeader(secBrowsingTopics string) ([]Topic, []error) {
	topics := make([]Topic, 0, 10)
	var warnings []error

	for _, field := range strings.Split(secBrowsingTopics, ",") {
		field = strings.TrimSpace(field)
		if field == "" || strings.HasPrefix(field, "();p=") {
			continue
		}

		if len(topics) < 10 {
			if topic, ok := parseTopicSegment(field); ok {
				topics = append(topics, topic)
			} else {
				warnings = append(warnings, formatWarning(field))
			}
		} else {
			warnings = append(warnings, formatWarning(field+" discarded due to limit reached."))
		}
	}

	return topics, warnings
}

// parseTopicSegment parses a single topic segment from the header into Topics object
func parseTopicSegment(field string) (Topic, bool) {
	segment := strings.Split(field, ";")
	if len(segment) != 2 {
		return Topic{}, false
	}

	segmentsIDs := strings.TrimSpace(segment[0])
	if len(segmentsIDs) < 3 || segmentsIDs[0] != '(' || segmentsIDs[len(segmentsIDs)-1] != ')' {
		return Topic{}, false
	}

	segtax, segclass := parseSegTaxSegClass(segment[1])
	if segtax == 0 || segclass == "" {
		return Topic{}, false
	}

	segIDs, err := parseSegmentIDs(segmentsIDs[1 : len(segmentsIDs)-1])
	if err != nil {
		return Topic{}, false
	}

	return Topic{
		SegTax:   segtax,
		SegClass: segclass,
		SegIDs:   segIDs,
	}, true
}

func parseSegTaxSegClass(seg string) (int, string) {
	taxanomyModel := strings.Split(seg, ":")
	if len(taxanomyModel) != 3 {
		return 0, ""
	}

	// taxanomyModel[0] is v=browser_version, we don't need it
	taxanomyVer := strings.TrimSpace(taxanomyModel[1])
	taxanomy, err := strconv.Atoi(taxanomyVer)
	if err != nil || taxanomy < 1 || taxanomy > 10 {
		return 0, ""
	}

	segtax := 600 + (taxanomy - 1)
	segclass := strings.TrimSpace(taxanomyModel[2])
	return segtax, segclass
}

// parseSegmentIDs parses the segment ids from the header string into int array
func parseSegmentIDs(segmentsIDs string) ([]int, error) {
	var selectedSegmentIDs []int
	for _, segmentID := range strings.Fields(segmentsIDs) {
		segmentID = strings.TrimSpace(segmentID)
		selectedSegmentID, err := strconv.Atoi(segmentID)
		if err != nil || selectedSegmentID <= 0 {
			return selectedSegmentIDs, errors.New("invalid segment id")
		}
		selectedSegmentIDs = append(selectedSegmentIDs, selectedSegmentID)
	}

	return selectedSegmentIDs, nil
}

func UpdateUserDataWithTopics(userData []openrtb2.Data, headerData []Topic, topicsDomain string) []openrtb2.Data {
	if topicsDomain == "" {
		return userData
	}

	// headerDataMap groups segIDs by segtax and segclass for faster lookup and tracking of new segIDs yet to be added to user.data
	// tracking is done by removing segIDs from segIDsMap once they are added to user.data, ensuring that headerDataMap will always have unique segtax-segclass-segIDs
	// the only drawback of tracking via deleting segtax-segclass from headerDataMap is that this would not track duplicate entries within user.data which is fine because we are only merging header data with the provided user.data
	headerDataMap := createHeaderDataMap(headerData)

	for i, data := range userData {
		ext := &Topic{}
		err := json.Unmarshal(data.Ext, ext)
		if err != nil {
			continue
		}

		if ext.SegTax == 0 || ext.SegClass == "" {
			continue
		}

		if newSegIDs := findNewSegIDs(data.Name, topicsDomain, *ext, data.Segment, headerDataMap); newSegIDs != nil {
			for _, segID := range newSegIDs {
				userData[i].Segment = append(userData[i].Segment, openrtb2.Segment{ID: strconv.Itoa(segID)})
			}

			delete(headerDataMap[ext.SegTax], ext.SegClass)
		}
	}

	for segTax, segClassMap := range headerDataMap {
		for segClass, segIDs := range segClassMap {
			if len(segIDs) != 0 {
				data := openrtb2.Data{
					Name: topicsDomain,
				}

				var err error
				data.Ext, err = jsonutil.Marshal(Topic{SegTax: segTax, SegClass: segClass})
				if err != nil {
					continue
				}

				for segID := range segIDs {
					data.Segment = append(data.Segment, openrtb2.Segment{
						ID: strconv.Itoa(segID),
					})
				}

				userData = append(userData, data)
			}
		}
	}

	return userData
}

// createHeaderDataMap creates a map of header data (segtax-segclass-segIDs) for faster lookup
// topicsdomain is not needed as we are only interested data from one domain configured in host config
func createHeaderDataMap(headerData []Topic) map[int]map[string]map[int]struct{} {
	headerDataMap := make(map[int]map[string]map[int]struct{})

	for _, topic := range headerData {
		segClassMap, ok := headerDataMap[topic.SegTax]
		if !ok {
			segClassMap = make(map[string]map[int]struct{})
			headerDataMap[topic.SegTax] = segClassMap
		}

		segIDsMap, ok := segClassMap[topic.SegClass]
		if !ok {
			segIDsMap = make(map[int]struct{})
			segClassMap[topic.SegClass] = segIDsMap
		}

		for _, segID := range topic.SegIDs {
			segIDsMap[segID] = struct{}{}
		}
	}

	return headerDataMap
}

// findNewSegIDs merge unique segIDs in single user.data if request.user.data and header data match. i.e. segclass, segtax and topicsdomain match
func findNewSegIDs(dataName, topicsDomain string, userData Topic, userDataSegments []openrtb2.Segment, headerDataMap map[int]map[string]map[int]struct{}) []int {
	if dataName != topicsDomain {
		return nil
	}

	segClassMap, exists := headerDataMap[userData.SegTax]
	if !exists {
		return nil
	}

	segIDsMap, exists := segClassMap[userData.SegClass]
	if !exists {
		return nil
	}

	// remove existing segIDs entries
	for _, segID := range userDataSegments {
		if id, err := strconv.Atoi(segID.ID); err == nil {
			delete(segIDsMap, id)
		}
	}

	// collect remaining segIDs
	segIDs := make([]int, 0, len(segIDsMap))
	for segID := range segIDsMap {
		segIDs = append(segIDs, segID)
	}

	return segIDs
}

func formatWarning(msg string) error {
	return &errortypes.DebugWarning{
		WarningCode: errortypes.SecBrowsingTopicsWarningCode,
		Message:     fmt.Sprintf("Invalid field in Sec-Browsing-Topics header: %s", msg),
	}
}
