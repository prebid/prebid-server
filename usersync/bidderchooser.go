package usersync

// bidderChooser determines which bidders to consider for user syncing.
type bidderChooser interface {
	// choose returns an ordered collection of potentially non-unique bidders.
	choose(requested, available []string, cooperative Cooperative) []string
}

// standardBidderChooser implements the bidder choosing algorithm per official Prebid specification.
type standardBidderChooser struct {
	shuffler shuffler
}

func (c standardBidderChooser) choose(requested, available []string, cooperative Cooperative) []string {
	if len(requested) == 0 {
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
