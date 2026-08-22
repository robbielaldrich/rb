package catalog

// Card is a single Riftbound card, as returned by the Riftcodex API
// (https://riftcodex.com/docs/endpoints/cards/) and stored in cards.json.
type Card struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	RiftboundID     string         `json:"riftbound_id"`
	TCGPlayerID     *string        `json:"tcgplayer_id"`
	CollectorNumber int            `json:"collector_number"`
	Attributes      Attributes     `json:"attributes"`
	Classification  Classification `json:"classification"`
	Text            Text           `json:"text"`
	Set             CardSet        `json:"set"`
	Media           Media          `json:"media"`
	Tags            []string       `json:"tags"`
	Orientation     string         `json:"orientation"`
	Metadata        Metadata       `json:"metadata"`
	New             *bool          `json:"new"`
}

// Attributes holds a card's numeric stats. Any of these may be absent
// depending on the card's type (e.g. only Units have Might).
type Attributes struct {
	Energy *int `json:"energy"`
	Might  *int `json:"might"`
	Power  *int `json:"power"`
}

// Classification describes what kind of card this is.
type Classification struct {
	Type      string   `json:"type"`
	Supertype *string  `json:"supertype"`
	Rarity    string   `json:"rarity"`
	Domain    []string `json:"domain"`
}

// Text holds a card's rules text in various forms.
type Text struct {
	Rich    string  `json:"rich"`
	Plain   string  `json:"plain"`
	Flavour *string `json:"flavour"`
}

// CardSet identifies the set a card belongs to.
type CardSet struct {
	SetID string `json:"set_id"`
	Label string `json:"label"`
}

// Media holds a card's image and artist information.
type Media struct {
	ImageURL          string `json:"image_url"`
	Artist            string `json:"artist"`
	AccessibilityText string `json:"accessibility_text"`
}

// Metadata holds bookkeeping information about a card.
type Metadata struct {
	CleanName    *string `json:"clean_name"`
	UpdatedOn    string  `json:"updated_on"`
	AlternateArt bool    `json:"alternate_art"`
	Overnumbered bool    `json:"overnumbered"`
	Signature    bool    `json:"signature"`
}
