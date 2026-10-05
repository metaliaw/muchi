package directory

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
	config, err := LoadStoreConfig("../../../config/stores.yaml", nil)
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
	config, err := LoadStoreConfig("../../../config/stores.yaml", nil)
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

// Los Idiomas se Declaran en la Configuración y viajan con el Juego: quien
// Dibuja el Campo no Adivina qué Nombres entiende el Catálogo.
func TestTheGameCarriesTheLanguagesItDeclares(t *testing.T) {
	config, err := LoadStoreConfig("../../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}

	games := SupportedGames(config)

	magic := supportOf(t, games, "magic")
	if len(magic.Languages) != 2 || magic.Languages[0].Code != "es" || magic.Languages[1].Code != "en" {
		t.Fatalf("magic languages=%+v", magic.Languages)
	}
	if magic.Languages[0].Label != "Español" || magic.Languages[1].Example != "Sol Ring" {
		t.Fatalf("magic languages=%+v", magic.Languages)
	}
	if pokemon := supportOf(t, games, "pokemon"); len(pokemon.Languages) != 0 {
		t.Fatalf("a game with no declaration announced languages: %+v", pokemon.Languages)
	}
	if declared := SupportedLanguages(config); len(declared["magic"]) != 2 || declared["pokemon"] != nil {
		t.Fatalf("declared=%v", declared)
	}
}

func TestTheConfigRefusesLanguagesItCannotServe(t *testing.T) {
	catalog := map[string]LanguageConfig{"es": {Label: "Español", Example: "Anillo solar"}}
	base := func(languages GameLanguages) Config {
		config, err := LoadStoreConfig("../../../config/stores.yaml", nil)
		if err != nil {
			t.Fatal(err)
		}
		config.Languages = catalog
		magic := config.Games["magic"]
		magic.Languages = languages
		config.Games["magic"] = magic
		return config
	}
	for name, languages := range map[string]GameLanguages{
		"a language the catalog does not know": {Provider: "scryfall", Supported: []string{"es", "ja"}},
		"a provider with no adapter":           {Provider: "inventado", Supported: []string{"es"}},
		"a provider with no language":          {Provider: "scryfall"},
		"languages with no provider":           {Supported: []string{"es"}},
	} {
		if err := ValidateConfig(base(languages)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := ValidateConfig(base(GameLanguages{Provider: "scryfall", Supported: []string{"es"}})); err != nil {
		t.Errorf("a declared language was refused: %v", err)
	}
}
