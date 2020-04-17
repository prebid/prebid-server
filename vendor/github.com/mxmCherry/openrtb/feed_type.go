package openrtb

// 5.16 Feed Types
//
// Types of feeds, typically for audio.
type FeedType int8

const (
	FeedTypeMusicService  FeedType = 1 // Music Service
	FeedTypeFMAMBroadcast FeedType = 2 // FM/AM Broadcast
	FeedTypePodcast       FeedType = 3 // Podcast
)
