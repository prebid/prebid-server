package usersync

import (
	"errors"
	"math"
	"time"
)

type Ejector interface {
	Choose(uids map[string]UIDEntry) (string, error)
}

type OldestEjector struct {
	nonPriorityKeys []string
	PriorityGroups  [][]string
	EjectPriority   bool
}

type PriorityBidderEjector struct {
	PriorityGroups   [][]string
	SyncerKey        string
	OldestEjector    OldestEjector
	IsSyncerPriority bool
}

// Choose method for oldest ejector will return the oldest uid after determing which set of elements to be choosing from
func (o *OldestEjector) Choose(uids map[string]UIDEntry) (string, error) {
	var elements []string

	// Set which elements the ejector will be choosing from
	if len(o.nonPriorityKeys) > 0 {
		elements = o.nonPriorityKeys
	} else if o.EjectPriority {
		elements = o.PriorityGroups[len(o.PriorityGroups)-1]
	} else {
		elements = getNonPriorityKeys(uids, o.PriorityGroups)
	}

	uidToDelete := getOldestElement(elements, uids)

	return uidToDelete, nil
}

// Choose method for priority ejector will return the oldest lowest priority element
func (p *PriorityBidderEjector) Choose(uids map[string]UIDEntry) (string, error) {
	p.OldestEjector.nonPriorityKeys = getNonPriorityKeys(uids, p.PriorityGroups)

	if len(p.OldestEjector.nonPriorityKeys) > 0 {
		p.OldestEjector.EjectPriority = false
		return p.OldestEjector.Choose(uids)
	}

	if p.IsSyncerPriority {
		lowestPriorityGroup := p.PriorityGroups[len(p.PriorityGroups)-1]

		if len(lowestPriorityGroup) == 1 {
			uidToDelete := lowestPriorityGroup[0]
			p.PriorityGroups = removeElementFromPriorityGroup(p.PriorityGroups, uidToDelete)
			return uidToDelete, nil
		}

		p.OldestEjector.EjectPriority = true
		p.OldestEjector.PriorityGroups = p.PriorityGroups
		uidToDelete, err := p.OldestEjector.Choose(uids)
		if err != nil {
			return "", err
		}
		p.PriorityGroups = removeElementFromPriorityGroup(p.PriorityGroups, uidToDelete)
		return uidToDelete, nil
	}
	return "", errors.New("syncer key " + p.SyncerKey + " is not in priority groups")
}

// updatePriorityGroup will remove the selected element from the priority groups, and will remove the entire priority group if it's empty
func removeElementFromPriorityGroup(priorityGroups [][]string, oldestElement string) [][]string {
	lowestPriorityGroup := priorityGroups[len(priorityGroups)-1]
	if len(lowestPriorityGroup) <= 1 {
		priorityGroups = priorityGroups[:len(priorityGroups)-1]
		return priorityGroups
	}

	for index, elem := range lowestPriorityGroup {
		if elem == oldestElement {
			updatedPriorityGroup := append(lowestPriorityGroup[:index], lowestPriorityGroup[index+1:]...)
			priorityGroups[len(priorityGroups)-1] = updatedPriorityGroup
			return priorityGroups
		}
	}
	return priorityGroups
}

func getNonPriorityKeys(uids map[string]UIDEntry, priorityGroups [][]string) []string {
	// If no priority groups, then all keys in uids are non-priority
	nonPriorityKeys := []string{}
	if len(priorityGroups) == 0 {
		for key := range uids {
			nonPriorityKeys = append(nonPriorityKeys, key)
		}
		return nonPriorityKeys
	}

	// Create map of keys that are a priority
	isPriority := make(map[string]bool)
	for _, group := range priorityGroups {
		for _, bidder := range group {
			isPriority[bidder] = true
		}
	}

	// Loop over uids and compare the keys in this map to the keys in the isPriority map to find the non piority keys
	for key := range uids {
		if !isPriority[key] {
			nonPriorityKeys = append(nonPriorityKeys, key)
		}
	}

	return nonPriorityKeys
}

func getOldestElement(elements []string, uids map[string]UIDEntry) string {
	var oldestElem string
	var oldestDate int64 = math.MaxInt64

	for _, key := range elements {
		if value, ok := uids[key]; ok {
			timeUntilExpiration := time.Until(value.Expires)
			if timeUntilExpiration < time.Duration(oldestDate) {
				oldestElem = key
				oldestDate = int64(timeUntilExpiration)
			}
		}
	}
	return oldestElem
}
