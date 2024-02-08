package privacysandbox

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type Topic struct {
	SegTax   int
	SegClass string
	SegIDs   []int
}

// ParseTopicsFromHeader parses the Sec-Browsing-Topics header data into Topics object
func ParseTopicsFromHeader(secBrowsingTopics string) []Topic {
	var topics []Topic

	for _, seg := range strings.Split(secBrowsingTopics, ",") {
		if topic, ok := parseTopicSegment(seg); ok {
			topics = append(topics, topic)
		}

		if len(topics) == 10 {
			break
		}
	}

	return topics
}

// parseTopicSegment parses a single topic segment from the header into Topics object
func parseTopicSegment(seg string) (Topic, bool) {
	seg = strings.TrimSpace(seg)
	if seg == "" || strings.HasPrefix(seg, "();p=") {
		return Topic{}, false
	}

	segment := strings.Split(seg, ";")
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

	return Topic{
		SegTax:   segtax,
		SegClass: segclass,
		SegIDs:   parseSegmentIDs(segmentsIDs[1 : len(segmentsIDs)-1]),
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
func parseSegmentIDs(segmentsIDs string) []int {
	var selectedSegmentIDs []int
	for _, segmentID := range strings.Fields(segmentsIDs) {
		segmentID = strings.TrimSpace(segmentID)
		if selectedSegmentID, err := strconv.Atoi(segmentID); err == nil && selectedSegmentID > 0 {
			selectedSegmentIDs = append(selectedSegmentIDs, selectedSegmentID)
		}
	}

	return selectedSegmentIDs
}

func UpdateUserDataWithTopics(userData []openrtb2.Data, headerData []Topic, topicsDomain string) []openrtb2.Data {
	if topicsDomain == "" {
		return userData
	}

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

		if newSegIDs := mergeSegIDs(data.Name, topicsDomain, *ext, data.Segment, headerDataMap); newSegIDs != nil {
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
					Ext:  json.RawMessage(fmt.Sprintf(`{"segtax": %d, "segclass": "%s"}`, segTax, segClass)),
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
		if _, ok := headerDataMap[topic.SegTax]; !ok {
			headerDataMap[topic.SegTax] = make(map[string]map[int]struct{})
		}
		if _, ok := headerDataMap[topic.SegTax][topic.SegClass]; !ok {
			headerDataMap[topic.SegTax][topic.SegClass] = make(map[int]struct{})
		}
		for _, segID := range topic.SegIDs {
			headerDataMap[topic.SegTax][topic.SegClass][segID] = struct{}{}
		}
	}

	return headerDataMap
}

// mergeSegIDs merge unique segIDs in single user.data if request.user.data and header data match. i.e. segclass, segtax and topicsdomain match
func mergeSegIDs(dataName, topicsDomain string, userData Topic, userDataSegments []openrtb2.Segment, headerDataMap map[int]map[string]map[int]struct{}) []int {
	if dataName == topicsDomain {
		if _, ok := headerDataMap[userData.SegTax]; ok {
			if _, ok := headerDataMap[userData.SegTax][userData.SegClass]; ok {

				for _, segID := range userDataSegments {
					if id, err := strconv.Atoi(segID.ID); err == nil {
						delete(headerDataMap[userData.SegTax][userData.SegClass], id)
					}
				}

				var segIDs []int
				for segID := range headerDataMap[userData.SegTax][userData.SegClass] {
					segIDs = append(segIDs, segID)
				}
				return segIDs
			}
		}
	}

	return nil
}
