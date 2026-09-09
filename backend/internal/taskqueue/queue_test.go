package taskqueue

import (
	"regexp"
	"testing"
)

// Cloud Tasks admite Letras, Números, Guiones y Bajos. Un Punto rompe la
// Llamada con InvalidArgument, y el Barrido entero se cae con ella.
var legalName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestAWakeNameIsLegalForCloudTasks(t *testing.T) {
	name := buildWakeName("sweep")
	if !legalName.MatchString(name) {
		t.Fatalf("Cloud Tasks rechazaría este Nombre: %q", name)
	}
}

func TestTwoWakesInTheSameSecondDoNotCollide(t *testing.T) {
	// Un Barrido pide muchos Despertares seguidos. Con Nombres repetidos,
	// Cloud Tasks contesta AlreadyExists, dispatch se lo traga, y el Barredor
	// repone uno solo creyendo que repuso todos.
	seen := map[string]bool{}
	for range 500 {
		name := buildWakeName("sweep")
		if seen[name] {
			t.Fatalf("dos Despertares comparten Nombre: %q", name)
		}
		seen[name] = true
	}
}
