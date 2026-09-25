package stores

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// pageFetcher Serves one Storefront Page, the Way a Buyer would Read it.
type pageFetcher struct{ body string }

func (f pageFetcher) FetchSource(_ context.Context, _, _ string) ([]byte, error) {
	return []byte(f.body), nil
}

func readUnits(t *testing.T, body string, config StoreConfig) model.StockReading {
	t.Helper()
	config.Enabled = true
	checker := Checker{Fetcher: pageFetcher{body}, Config: Config{
		Stores: map[string]StoreConfig{"tienda.test": config}}}
	reading, err := checker.CheckStock(context.Background(),
		model.Offer{URL: "https://tienda.test/carta/sol-ring"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	return reading
}

func TestCountsTheUnitsThePageDeclares(t *testing.T) {
	// Lo que scry Escribe junto al Campo de Cantidad, tal cual se Lee.
	reading := readUnits(t, `<p class="help_stock">3 disponibles.</p>`, StoreConfig{})
	if reading.Status != "available" || reading.Quantity == nil || *reading.Quantity != 3 {
		t.Fatalf("status=%s quantity=%v", reading.Status, reading.Quantity)
	}
}

func TestASilentPageStaysASilentPage(t *testing.T) {
	// Inventar un Número Volvería un Silencio en una Promesa.
	reading := readUnits(t, `<p>Sol Ring</p>`, StoreConfig{})
	if reading.Status != "unknown" || reading.Quantity != nil {
		t.Fatalf("status=%s quantity=%v", reading.Status, reading.Quantity)
	}
}

func TestZeroUnitsIsSoldOut(t *testing.T) {
	reading := readUnits(t, `<p>0 disponibles</p>`, StoreConfig{})
	if reading.Status != "unavailable" || reading.Quantity == nil || *reading.Quantity != 0 {
		t.Fatalf("status=%s quantity=%v", reading.Status, reading.Quantity)
	}
}

func TestSoldOutTextBeatsTheCount(t *testing.T) {
	// Una Ficha que Dice Agotado y a la vez Cuenta una Unidad no Vende: el
	// Número Suele ser el Resto de una Plantilla, y el Cartel es la Respuesta.
	reading := readUnits(t, `<p>Agotado</p><p>1 disponible</p>`,
		StoreConfig{UnavailableText: []string{"Agotado"}})
	if reading.Status != "unavailable" || reading.Quantity == nil || *reading.Quantity != 0 {
		t.Fatalf("status=%s quantity=%v", reading.Status, reading.Quantity)
	}
}
