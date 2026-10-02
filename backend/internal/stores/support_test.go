package stores

import "testing"

func supportOf(t *testing.T, games []GameSupport, key string) GameSupport {
	t.Helper()
	for _, game := range games {
		if game.Key == key {
			return game
		}
	}
	t.Fatalf("game missing: %s", key)
	return GameSupport{}
}

// Nadie Declara qué Soporta un Juego: lo Declara cada Tienda, y el Juego Hereda
// la Suma. Esto se Mide contra la Configuración real, que es la que Manda.
func TestTheGameInheritsWhatItsSourcesSell(t *testing.T) {
	config, err := LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}

	games := SupportedGames(config)

	// Mitos y Leyendas Llega por dos Mitades: el Marketplace Pone las Cartas y
	// Casa MyL las Cajas. Ninguna de las dos Cubre el Juego sola.
	mitos := supportOf(t, games, "mitos-y-leyendas")
	if !mitos.Singles || !mitos.Sealed {
		t.Fatalf("mitos=%+v", mitos)
	}
	magic := supportOf(t, games, "magic")
	if !magic.Singles || !magic.Sealed {
		t.Fatalf("magic=%+v", magic)
	}
}

func TestAGameWithoutSourcesIsNotOffered(t *testing.T) {
	config := Config{
		Games: map[string]GameConfig{
			"solo": {Name: "Solo", Enabled: true, Origins: []string{"shopify"}},
		},
	}

	if games := SupportedGames(config); len(games) != 0 {
		t.Fatalf("games=%+v", games)
	}
}

func TestASealedOnlyStoreLeavesTheGameWithoutCards(t *testing.T) {
	no := false
	config := Config{
		Games: map[string]GameConfig{
			"cajas": {Name: "Cajas", Enabled: true, Origins: []string{"shopify"}},
		},
		Stores: map[string]StoreConfig{
			"cajas.test": {Games: []string{"cajas"}, Platform: "shopify", Enabled: true, Singles: &no},
		},
	}

	games := SupportedGames(config)

	game := supportOf(t, games, "cajas")
	if game.Singles || !game.Sealed {
		t.Fatalf("game=%+v", game)
	}
	if game.Asked(false) || !game.Asked(true) {
		t.Fatalf("asked singles=%v sealed=%v", game.Asked(false), game.Asked(true))
	}
}

// Una Tienda que no se Busca no Cuenta: Existe para Comprobar su Stock.
func TestAStoreThatIsNotSearchedDoesNotSupportAGame(t *testing.T) {
	no := false
	config := Config{
		Games: map[string]GameConfig{
			"solo": {Name: "Solo", Enabled: true, Origins: []string{"shopify"}},
		},
		Stores: map[string]StoreConfig{
			"comprobable.test": {Games: []string{"solo"}, Platform: "shopify", Enabled: true, Searched: &no},
		},
	}

	if games := SupportedGames(config); len(games) != 0 {
		t.Fatalf("games=%+v", games)
	}
}

func TestListedStoresKeepStreetAndCoordinates(t *testing.T) {
	config, err := LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, listed := range ListedStores(config) {
		if listed.ID != "www.paytowin.cl" {
			continue
		}
		location := listed.Locations[0]
		if location.Street == "" || location.Latitude == nil || location.Longitude == nil {
			t.Fatalf("location=%+v", location)
		}
		return
	}
	t.Fatal("Pay to Win not listed")
}

func TestHalfACoordinatePairIsRefused(t *testing.T) {
	latitude := -33.4
	if err := validateLocations([]StoreLocation{{City: "Las Condes", Latitude: &latitude}}); err == nil {
		t.Fatal("a latitude without longitude was accepted")
	}
}
