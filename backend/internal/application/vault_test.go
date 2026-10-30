package application

import (
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
)

// Estas Afirmaciones Fijan la Frontera. Un Almacén nuevo Compila el Día que
// Cumple estos Puertos, y uno que Deja de Cumplirlos Falla acá y no en Producción.
var (
	_ Vault = (*repositories.SearchOffer)(nil)
)
