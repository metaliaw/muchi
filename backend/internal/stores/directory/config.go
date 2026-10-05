// Package directory Reads stores.yaml and Answers what the Directory Lists:
// the Games, the Search Providers and the Stores, with the Views a Picker and a
// Selector Need. It Names no Platform Client; commerce Opens those.
package directory

type Config struct {
	// Languages Names every Language a Game may Declare: how a Person Reads it
	// and how a Card Name Looks written in it. The Provider Knows the Code; the
	// Label and the Example are for whoever Draws the Box.
	Languages       map[string]LanguageConfig       `yaml:"languages"`
	Games           map[string]GameConfig           `yaml:"games"`
	SearchProviders map[string]SearchProviderConfig `yaml:"search_providers"`
	Stores          map[string]StoreConfig          `yaml:"stores"`
}

type GameConfig struct {
	Name         string        `yaml:"name"`
	ReferenceKey string        `yaml:"reference_key"`
	Enabled      bool          `yaml:"enabled"`
	Origins      []string      `yaml:"origins"`
	Languages    GameLanguages `yaml:"languages"`
}

// LanguageConfig Is one Entry of the Language Catalog.
type LanguageConfig struct {
	Label   string `yaml:"label"`
	Example string `yaml:"example"`
}

// GameLanguages Says who Reads Card Names for a Game and in which Languages.
// Absent Means the Game Declares none: its Names are Asked as Written, and any
// Language a Caller Names is Passed along untouched.
type GameLanguages struct {
	// Provider Names the Catalog that Answers the Game's Card Names, one of
	// LanguageProviders.
	Provider  string   `yaml:"provider"`
	Supported []string `yaml:"supported"`
}

// Declared Answers whether the Game Names its Languages at all.
func (g GameLanguages) Declared() bool {
	return g.Provider != "" || len(g.Supported) > 0
}

type SearchProviderConfig struct {
	Type      string   `yaml:"type"`
	URL       string   `yaml:"url"`
	Enabled   bool     `yaml:"enabled"`
	Games     []string `yaml:"games"`
	Stores    []string `yaml:"stores"`
	Community bool     `yaml:"community"`
	// Sealed Says whether this Aggregator Indexes Unopened Product. An Index
	// built from Singles Answers a Sealed Question with nothing, or with the
	// Card Printed inside the Box, and either way it Spends a Request and Waits
	// for it. Absent Means yes, so a Source Stays Asked until someone Looks.
	Sealed *bool `yaml:"sealed"`
	// Singles Says whether this Aggregator Indexes Loose Cards. Absent Means
	// yes, for the same Reason as Sealed.
	Singles *bool `yaml:"singles"`
}

// ServesSingles Answers whether a Card Question is Worth Asking this Provider.
func (c SearchProviderConfig) ServesSingles() bool {
	return c.Singles == nil || *c.Singles
}

// ServesSealed Answers whether a Sealed Question is Worth Asking this Provider.
func (c SearchProviderConfig) ServesSealed() bool {
	return c.Sealed == nil || *c.Sealed
}

type StoreConfig struct {
	Games                    []string        `yaml:"games"`
	Platform                 string          `yaml:"platform"`
	Locations                []StoreLocation `yaml:"locations"`
	UnavailableSelectors     []string        `yaml:"unavailable_selectors"`
	UnavailableText          []string        `yaml:"unavailable_text"`
	ScopeSelector            string          `yaml:"scope_selector"`
	TimeoutSeconds           int             `yaml:"timeout_seconds"`
	EstimatedResponseSeconds int             `yaml:"estimated_response_seconds"`
	AllowRedirects           bool            `yaml:"allow_redirects"`
	Enabled                  bool            `yaml:"enabled"`
	Name                     string          `yaml:"name"`
	Lists                    []MoxfieldList  `yaml:"lists"`
	// Searched Says whether we Ask this Store for Offers ourselves. A Store
	// that Reaches us through a Search Provider is already Represented: the
	// Entry Exists so its Stock can be Checked, not so it is Asked twice.
	// Absent Means yes.
	Searched *bool `yaml:"searched"`
	// Singles Says whether this Store Sells Loose Cards at all. A Shop that
	// Sells only Boxes Answers a Card Name with a Box, and that Box Reads like
	// an Offer for the Card. Absent Means yes.
	Singles *bool `yaml:"singles"`
	// Sealed Says whether this Store Sells Unopened Product at all. A Singles
	// Shop Answers every Sealed Question empty, and the Answer Comes back
	// Marked incomplete when it Times out first. Absent Means yes.
	Sealed *bool `yaml:"sealed"`
}

// IsSearched Answers whether this Store is a Source of its own.
func (c StoreConfig) IsSearched() bool {
	return c.Searched == nil || *c.Searched
}

// SellsSingles Answers whether a Card Question is Worth Asking this Store.
func (c StoreConfig) SellsSingles() bool {
	return c.Singles == nil || *c.Singles
}

// SellsSealed Answers whether a Sealed Question is Worth Asking this Store.
func (c StoreConfig) SellsSealed() bool {
	return c.Sealed == nil || *c.Sealed
}

type StoreLocation struct {
	Country  string `yaml:"country"`
	Region   string `yaml:"region"`
	City     string `yaml:"city"`
	District string `yaml:"district"`
	Pickup   string `yaml:"pickup"`
	// Street Names the Door and Latitude with Longitude the Spot, both as
	// the Store Publishes them. Coordinates Come as a Pair or not at all.
	Street    string   `yaml:"street"`
	Latitude  *float64 `yaml:"latitude"`
	Longitude *float64 `yaml:"longitude"`
}

// MoxfieldList Prices a public https://moxfield.com deck as store inventory.
type MoxfieldList struct {
	Label       string `yaml:"label"`
	URL         string `yaml:"url"`
	CLPPerCKUSD int    `yaml:"clp_per_ck_usd"`
}
