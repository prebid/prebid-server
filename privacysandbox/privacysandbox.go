package privacysandbox

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
)

const (
	topicSegmentSeparator = ","
	topicMaxLength        = 10
)

type Topic struct {
	SegTax   int
	SegClass string
	SegIDs   []int
}

// ParseTopicsFromHeader parses the Sec-Browsing-Topics header data into Topics object
func ParseTopicsFromHeader(secBrowsingTopics string) []Topic {
	var topics []Topic

	for _, seg := range strings.Split(secBrowsingTopics, topicSegmentSeparator) {
		if topic, ok := parseTopicSegment(seg); ok {
			topics = append(topics, topic)
		}

		if len(topics) == topicMaxLength {
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

	segmentsIds := strings.TrimSpace(segment[0])
	if len(segmentsIds) < 3 || segmentsIds[0] != '(' || segmentsIds[len(segmentsIds)-1] != ')' {
		return Topic{}, false
	}

	segtax, segclass := parseSegTaxSegClass(segment[1])
	if segtax == 0 || segclass == "" {
		return Topic{}, false
	}

	return Topic{
		SegTax:   segtax,
		SegClass: segclass,
		SegIDs:   parseSegmentIDs(segmentsIds[1 : len(segmentsIds)-1]),
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
func parseSegmentIDs(segmentsIds string) []int {
	var segIDs []int
	for _, segId := range strings.Fields(segmentsIds) {
		segId = strings.TrimSpace(segId)
		if segid, err := strconv.Atoi(segId); err == nil && segid > 0 {
			segIDs = append(segIDs, segid)
		}
	}

	return segIDs
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
			for _, segId := range newSegIDs {
				userData[i].Segment = append(userData[i].Segment, openrtb2.Segment{ID: strconv.Itoa(segId)})
			}

			delete(headerDataMap[ext.SegTax], ext.SegClass)
		}
	}

	for segtax, segClassMap := range headerDataMap {
		for segclass, segIds := range segClassMap {
			if len(segIds) != 0 {
				data := openrtb2.Data{
					Name: topicsDomain,
					Ext:  json.RawMessage(fmt.Sprintf(`{"segtax": %d, "segclass": "%s"}`, segtax, segclass)),
				}

				for segId := range segIds {
					data.Segment = append(data.Segment, openrtb2.Segment{
						ID: strconv.Itoa(segId),
					})
				}

				userData = append(userData, data)
			}
		}
	}

	return userData
}

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

				var segIds []int
				for segID := range headerDataMap[userData.SegTax][userData.SegClass] {
					segIds = append(segIds, segID)
				}
				return segIds
			}
		}
	}

	return nil
}
