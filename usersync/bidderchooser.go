package usersync

import "github.com/prebid/prebid-server/config"

type bidderChooser interface {
	choose(requested, available []string, cooperative config.UserSyncCooperative) []string
}

type randomBidderChooser struct {
	shuffler shuffler
}

func (c randomBidderChooser) choose(requested, available []string, cooperative config.UserSyncCooperative) []string {
	if requested == nil {
		return c.shuffledCopy(available)
	}

	if cooperative.Enabled {
		return c.chooseCooperative(requested, available, cooperative.PriorityGroups)
	}

	return c.shuffledCopy(requested)
}

func (c randomBidderChooser) chooseCooperative(requested, available []string, priorityGroups [][]string) []string {
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

func (c randomBidderChooser) shuffledCopy(a []string) []string {
	if a == nil {
		return nil
	}
	aCopy := make([]string, len(a))
	copy(aCopy, a)
	c.shuffler.shuffle(aCopy)
	return aCopy
}

func (c randomBidderChooser) shuffledAppend(a, b []string) []string {
	startIndex := len(a)
	a = append(a, b...)
	c.shuffler.shuffle(a[startIndex:])
	return a
}
