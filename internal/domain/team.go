package domain

// TeamMetadata is the sport-agnostic record for a program the registry
// knows about. Not every NCAA scoreboard entry resolves to one — we
// only populate TeamMetadata for programs the sport package can
// identify by NCAA.com's `seo` slug.
type TeamMetadata struct {
	Seo        string   `json:"seo"`
	Name       string   `json:"name"`
	ShortName  string   `json:"short_name,omitempty"`
	SportID    SportID  `json:"sport_id"`
	Division   Division `json:"division"`
	Conference string   `json:"conference,omitempty"`
}
