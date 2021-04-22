package usersync

import "github.com/prebid/prebid-server/config"

type bidderChooser interface {
	// choose returns an ordered collection of potentially non-unique bidders.
	choose(requested, available []string, cooperative config.UserSyncCooperative) []string
}

type standardBidderChooser struct {
	shuffler shuffler
}

func (c standardBidderChooser) choose(requested, available []string, cooperative config.UserSyncCooperative) []string {
	if requested == nil {
		return c.shuffledCopy(available)
	}

	if cooperative.Enabled {
		return c.chooseCooperative(requested, available, cooperative.PriorityGroups)
	}

	return c.shuffledCopy(requested)
}

func (c standardBidderChooser) chooseCooperative(requested, available []string, priorityGroups [][]string) []string {
	biddersCapacity := int(float64(len(available)) * 1.5)
	bidders := make([]string, 0, biddersCapacity)

	// requested
	bidders = c.shuffledAppend(bidders, requested)

	// priority groups
	for _, group := range priorityGroups {
		bidders = c.shuffledAppend(bidders, group)
	}

	// available
	bidders = c.shuffledAppend(bidders, available)

	return bidders
}

func (c standardBidderChooser) shuffledCopy(a []string) []string {
	aCopy := make([]string, len(a))
	copy(aCopy, a)
	c.shuffler.shuffle(aCopy)
	return aCopy
}

func (c standardBidderChooser) shuffledAppend(a, b []string) []string {
	startIndex := len(a)
	a = append(a, b...)
	c.shuffler.shuffle(a[startIndex:])
	return a
}
