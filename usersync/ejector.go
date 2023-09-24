package usersync

import (
	"errors"
	"time"
)

type Ejector interface {
	Choose(uids map[string]UIDEntry) (string, error)
}

type OldestEjector struct{}

type PriorityBidderEjector struct {
	PriorityGroups   [][]string
	SyncersByBidder  map[string]Syncer
	IsSyncerPriority bool
	TieEjector       Ejector
}

// Choose method for oldest ejector will return the oldest uid
func (o *OldestEjector) Choose(uids map[string]UIDEntry) (string, error) {
	var oldestElement string
	var oldestDate time.Time = time.Unix(1<<63-62135596801, 999999999) // Max value for time

	for key, value := range uids {
		if value.Expires.Before(oldestDate) {
			oldestElement = key
			oldestDate = value.Expires
		}
	}
	return oldestElement, nil
}

// Choose method for priority ejector will return the oldest lowest priority element
func (p *PriorityBidderEjector) Choose(uids map[string]UIDEntry) (string, error) {
	nonPriorityUids := getNonPriorityUids(uids, p.PriorityGroups, p.SyncersByBidder)
	if err := p.checkSyncerPriority(nonPriorityUids); err != nil {
		return "", err
	}

	if len(nonPriorityUids) > 0 {
		return p.TieEjector.Choose(nonPriorityUids)
	}

	lowestPriorityGroup := p.PriorityGroups[len(p.PriorityGroups)-1]
	if len(lowestPriorityGroup) == 1 {
		uidToDelete := lowestPriorityGroup[0]
		p.PriorityGroups = removeElementFromPriorityGroup(p.PriorityGroups, uidToDelete)
		return uidToDelete, nil
	}

	lowestPriorityUids := getPriorityUids(lowestPriorityGroup, uids, p.SyncersByBidder)
	uidToDelete, err := p.TieEjector.Choose(lowestPriorityUids)
	if err != nil {
		return "", err
	}
	p.PriorityGroups = removeElementFromPriorityGroup(p.PriorityGroups, uidToDelete)
	return uidToDelete, nil
}

// updatePriorityGroup will remove the selected element from the priority groups, and will remove the entire priority group if it's empty
func removeElementFromPriorityGroup(priorityGroups [][]string, oldestElement string) [][]string {
	lowestPriorityGroup := priorityGroups[len(priorityGroups)-1]
	if len(lowestPriorityGroup) <= 1 {
		return priorityGroups[:len(priorityGroups)-1]
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

func getNonPriorityUids(uids map[string]UIDEntry, priorityGroups [][]string, syncersByBidder map[string]Syncer) map[string]UIDEntry {
	// If no priority groups, then all keys in uids are non-priority
	if len(priorityGroups) == 0 {
		return uids
	}

	// Create map of keys that are a priority
	isPriority := make(map[string]bool)
	for _, group := range priorityGroups {
		for _, bidder := range group {
			if bidderSyncer, ok := syncersByBidder[bidder]; ok {
				isPriority[bidderSyncer.Key()] = true
			}
		}
	}

	// Create a map for non-priority uids
	nonPriorityUIDs := make(map[string]UIDEntry)

	// Loop over uids and populate the nonPriorityUIDs map with non-priority keys
	for key, value := range uids {
		if _, found := isPriority[key]; !found {
			nonPriorityUIDs[key] = value
		}
	}

	return nonPriorityUIDs
}

func getPriorityUids(lowestPriorityGroup []string, uids map[string]UIDEntry, syncersByBidder map[string]Syncer) map[string]UIDEntry {
	lowestPriorityUIDs := make(map[string]UIDEntry)

	// Loop over lowestPriorityGroup and populate the lowestPriorityUIDs map
	for _, bidder := range lowestPriorityGroup {
		if bidderSyncer, ok := syncersByBidder[bidder]; ok {
			if uidEntry, exists := uids[bidderSyncer.Key()]; exists {
				lowestPriorityUIDs[bidderSyncer.Key()] = uidEntry
			}
		}
	}
	return lowestPriorityUIDs
}

func (p *PriorityBidderEjector) checkSyncerPriority(nonPriorityUids map[string]UIDEntry) error {
	if len(nonPriorityUids) == 1 && !p.IsSyncerPriority && len(p.PriorityGroups) > 0 {
		return errors.New("syncer key is not a priority, and there are only priority elements left")
	}
	return nil
}
