package catalog

import (
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

// StoreLocations Names each Store by the Places it Serves.
func (b Builder) StoreLocations() map[string][]string {
	result := make(map[string][]string)
	for domain, store := range b.Shops.Config.Stores {
		locations := formatStoreLocations(store.Locations)
		if len(locations) == 0 {
			continue
		}
		result[domain] = locations
		if store.Name != "" {
			result[store.Name] = locations
		}
	}
	return result
}

func formatStoreLocations(locations []directory.StoreLocation) []string {
	result := make([]string, 0, len(locations))
	for _, location := range locations {
		detail := location.District
		if detail == "" {
			detail = location.Pickup
		}
		result = append(result, strings.Join(removeEmpty([]string{location.City, detail}), " - "))
	}
	return result
}

func removeEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
