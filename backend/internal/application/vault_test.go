package application

import (
	firestorestore "github.com/cangrejometralleta/muchi-api/internal/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

// Estas Afirmaciones Fijan la Frontera. Un Almacén nuevo Compila el Día que
// Cumple estos Puertos, y uno que Deja de Cumplirlos Falla acá y no en Producción.
var (
	_ Vault      = (*firestorestore.Store)(nil)
	_ Dispatcher = (*taskqueue.Queue)(nil)
)
