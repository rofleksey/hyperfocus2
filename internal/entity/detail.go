package entity

// SampleDetail is a StreamSample joined with its streamer's display fields and
// its session's resolved vod id. It is the shape returned to handlers.
type SampleDetail struct {
	StreamSample
	Login           string
	DisplayName     string
	ProfileImageURL string
	VodID           *string
}

// SessionDetail is a StreamSession joined with streamer display fields.
type SessionDetail struct {
	StreamSession
	Login           string
	DisplayName     string
	ProfileImageURL string
}
