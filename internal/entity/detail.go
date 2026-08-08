package entity

// SampleDetail is a StreamSample joined with its streamer's display fields and
// its session's resolved vod id. It is the shape returned to handlers.
// FuzzyScore is populated only when a survivor-name search is active and holds
// the best fuzzy-match score (0..1) of the query against the survivor names.
type SampleDetail struct {
	StreamSample
	Login           string
	DisplayName     string
	ProfileImageURL string
	VodID           *string
	FuzzyScore      *float64
}

// SessionDetail is a StreamSession joined with streamer display fields.
type SessionDetail struct {
	StreamSession
	Login           string
	DisplayName     string
	ProfileImageURL string
}
