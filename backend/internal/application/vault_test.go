package application

import (
	"github.com/cangrejometralleta/muchi-api/internal/db"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

// Estas Afirmaciones Fijan la Frontera. Un Almacén nuevo Compila el Día que
// Cumple estos Puertos, y uno que Deja de Cumplirlos Falla acá y no en Producción.
var (
	_ repositories.Database = (*db.Store)(nil)
	_ Vault                 = (*repositories.SearchOffer)(nil)
	_ Dispatcher            = (*taskqueue.Queue)(nil)
)
