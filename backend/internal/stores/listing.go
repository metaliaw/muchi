package stores

import "sort"

// ListedLocation Is where a Store Stands, as far as its Configuration Knows.
type ListedLocation struct {
	Country   string   `json:"country,omitempty"`
	Region    string   `json:"region,omitempty"`
	City      string   `json:"city"`
	District  string   `json:"district,omitempty"`
	Pickup    string   `json:"pickup,omitempty"`
	Street    string   `json:"street,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// ListedStore Is one Store a Caller can Name, with the Games it Answers for.
type ListedStore struct {
	ID        string           `json:"id"`
	Name      string           `json:"name,omitempty"`
	Platform  string           `json:"platform"`
	Games     []string         `json:"games"`
	Locations []ListedLocation `json:"locations"`
}

// ListedStores Reduces the Enabled Stores to what a Map or a Picker Needs.
func ListedStores(config Config) []ListedStore {
	listed := make([]ListedStore, 0, len(config.Stores))
	for domain, store := range config.Stores {
		if !store.Enabled {
			continue
		}
		locations := make([]ListedLocation, 0, len(store.Locations))
		for _, location := range store.Locations {
			locations = append(locations, ListedLocation(location))
		}
		listed = append(listed, ListedStore{ID: domain, Name: store.Name, Platform: store.Platform, Games: append([]string{}, store.Games...), Locations: locations})
	}
	sort.Slice(listed, func(one, other int) bool { return listed[one].ID < listed[other].ID })
	return listed
}
