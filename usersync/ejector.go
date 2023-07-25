package usersync

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

type Ejector interface {
	Choose(uids map[string]UIDEntry) (string, error)
}

type OldestEjector struct {
	nonPriorityKeys []string
	PriorityGroups  [][]string
	FallbackEjector FallbackEjector
}

func (o *OldestEjector) Choose(uids map[string]UIDEntry) (string, error) {
	if len(o.nonPriorityKeys) == 0 {
		o.nonPriorityKeys = getNonPriorityKeys(uids, o.PriorityGroups)
	}

	oldestElement := getOldestElement(o.nonPriorityKeys, uids)
	if oldestElement == "" {
		return o.FallbackEjector.Choose(uids)
	}

	return oldestElement, nil
}

type PriorityBidderEjector struct {
	PriorityGroups  [][]string
	SyncerKey       string
	OldestEjector   OldestEjector
	FallbackEjector FallbackEjector
}

func (p *PriorityBidderEjector) Choose(uids map[string]UIDEntry) (string, error) {
	// If there are non priority keys left in the cookie.uids, we should eject one of those using the OldestEjector
	p.OldestEjector.nonPriorityKeys = getNonPriorityKeys(uids, p.PriorityGroups)
	if len(p.OldestEjector.nonPriorityKeys) > 0 {
		return p.OldestEjector.Choose(uids)
	}

	if isSyncerPriority(p.SyncerKey, p.PriorityGroups) {
		// Eject Oldest Element from Lowest Priority Group and Update Priority Group
		lowestPriorityGroup := p.PriorityGroups[len(p.PriorityGroups)-1]

		oldestElement := getOldestElement(lowestPriorityGroup, uids)
		if oldestElement == "" {
			return p.FallbackEjector.Choose(uids)
		}

		p.PriorityGroups = removeElementFromPriorityGroup(p.PriorityGroups, oldestElement)

		return oldestElement, nil
	}
	return "", errors.New("syncer key " + p.SyncerKey + " is not in priority groups")
}

// updatePriorityGroup will remove the selected element from the given priority group list
func removeElementFromPriorityGroup(priorityGroups [][]string, oldestElement string) [][]string {
	lowestPriorityGroup := priorityGroups[len(priorityGroups)-1]
	if len(lowestPriorityGroup) <= 1 {
		priorityGroups = priorityGroups[:len(priorityGroups)-1] // Remove entire priority group since it will be empty
		return priorityGroups
	}

	for index, elem := range lowestPriorityGroup {
		if elem == oldestElement {
			updatedPriorityGroup := append(lowestPriorityGroup[:index], lowestPriorityGroup[index+1:]...) // Delete only that element from the group
			priorityGroups[len(priorityGroups)-1] = updatedPriorityGroup
			return priorityGroups
		}
	}
	return priorityGroups
}

func isSyncerPriority(syncer string, priorityGroups [][]string) bool {
	for _, group := range priorityGroups {
		for _, bidder := range group {
			if syncer == bidder {
				return true
			}
		}
	}
	return false
}

func getNonPriorityKeys(uids map[string]UIDEntry, priorityGroups [][]string) []string {
	nonPriorityKeys := []string{}
	if len(priorityGroups) == 0 {
		for key := range uids {
			nonPriorityKeys = append(nonPriorityKeys, key)
		}
		return nonPriorityKeys
	}

	for key := range uids {
		isPriority := false
		for _, group := range priorityGroups {
			for _, bidder := range group {
				if key == bidder {
					isPriority = true
					break
				}
			}
		}
		if !isPriority {
			nonPriorityKeys = append(nonPriorityKeys, key)
		}
	}
	return nonPriorityKeys
}

func getOldestElement(list []string, uids map[string]UIDEntry) string {
	var oldestElem string = ""
	var oldestDate int64 = math.MaxInt64

	for _, key := range list {
		if value, ok := uids[key]; ok {
			timeUntilExpiration := time.Until(value.Expires)
			if timeUntilExpiration < time.Duration(oldestDate) {
				oldestElem = key
				oldestDate = int64(timeUntilExpiration)
			}
		} else {
			continue
		}
	}
	return oldestElem
}

type FallbackEjector struct{}

// Choose implemention for Fallback Ejector, selects a random uid to eject when the other two ejectors can't come up with a uid
func (f *FallbackEjector) Choose(uids map[string]UIDEntry) (string, error) {
	keys := make([]string, len(uids))

	i := 0
	for key := range uids {
		keys[i] = key
		i++
	}

	rand.Seed(time.Now().UnixNano())
	return keys[rand.Intn(len(keys))], nil
}
