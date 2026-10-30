package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// FieldSource Names the Indexed Field a Health Collection Sorts by.
const FieldSource = "source"

// sourcePacing Spaces two Calls to one Domain. It is a Floor, not a Promise:
// a Store that Asks for more Room Gets it through throttleCooldown.
const sourcePacing = 250 * time.Millisecond

// throttleCooldown Holds a Domain that Answered 429. The Store Said the Calls
// Arrive too Fast, so the next one Waits — that is the Answer it Asked for,
// and it is Kept per Domain, so one Store Slowing down Never Quiets another.
const throttleCooldown = 30 * time.Second

// circuitThreshold Counts the Falls in a row that Leave a Store unasked, and
// circuitCooldown the first Wait. circuitCeiling Caps the Doubling.
const (
	circuitThreshold = 5
	circuitCooldown  = time.Minute
	circuitCeiling   = time.Hour
)

// SourceStore Records how every Source has Answered, Paces the Traffic toward
// each one, and Answers for the Store's own Health.
type SourceStore struct {
	database    Database
	collections Collections
	pacing      time.Duration
	cooldown    time.Duration
}

// NewSourceStore Reads and Writes the Health through the Database's Collections.
func NewSourceStore(database Database) *SourceStore {
	return &SourceStore{database: database, collections: database.Collections()}
}

// PaceSources Tunes both Waits. Zero Keeps the Default, so a Caller may Set
// one and Leave the other alone.
func (b *SourceStore) PaceSources(pacing, cooldown time.Duration) {
	b.pacing, b.cooldown = pacing, cooldown
}

func (b *SourceStore) pace() time.Duration {
	if b.pacing <= 0 {
		return sourcePacing
	}
	return b.pacing
}

func (b *SourceStore) cool() time.Duration {
	if b.cooldown <= 0 {
		return throttleCooldown
	}
	return b.cooldown
}

// CheckHealth Reads the one Record a Health Check Asks for; a Missing one is
// Healthy, since Reaching the Store is the Point.
func (b *SourceStore) CheckHealth(ctx context.Context) error {
	_, err := b.collections.Health.Get(ctx, "ping")
	if errors.Is(err, ErrMissing) {
		return nil
	}
	return err
}

// ListSourceHealth Answers every Source Recorded, sorted by its Name.
func (b *SourceStore) ListSourceHealth(ctx context.Context) ([]model.SourceHealth, error) {
	return b.collections.SourceHealth.List(ctx, Query{OrderBy: FieldSource})
}

func (b *SourceStore) AwaitSource(ctx context.Context, domain string) error {
	health, _ := b.loadSource(ctx, b.collections, domain)
	if health.CircuitOpenUntil != nil && health.CircuitOpenUntil.After(time.Now().UTC()) {
		return fmt.Errorf("%w until %s", source.ErrCircuitOpen, health.CircuitOpenUntil.Format(time.RFC3339))
	}
	allowed, err := b.reserveTraffic(ctx, domain)
	if err != nil {
		return err
	}
	return waitUntil(ctx, allowed)
}

func (b *SourceStore) RecordSource(ctx context.Context, domain string, latency time.Duration, sourceErr error) error {
	// Un Freno Cambia el Ritmo, no la Salud: la Tienda Pidio Espacio y aqui se
	// le Da, antes de Anotar nada. Fallar al Espaciar no Pierde la Anotacion.
	if source.Throttled(sourceErr) {
		_ = b.delaySource(ctx, domain, b.cool())
	}
	return b.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		health, _ := b.loadSource(ctx, tx, domain)
		updateSource(&health, latency, sourceErr)
		return tx.SourceHealth.Save(ctx, domain, health)
	})
}

// loadSource Reads one Domain's Health, or a Fresh one when it has none yet.
func (b *SourceStore) loadSource(ctx context.Context, from Collections, domain string) (model.SourceHealth, error) {
	health, err := from.SourceHealth.Get(ctx, domain)
	if err != nil {
		return model.SourceHealth{Source: domain}, err
	}
	return health, nil
}

// delaySource Pushes one Domain's next Turn away, never Closer: two Frenos
// Arriving together must not Shorten the Wait the first one Won.
func (b *SourceStore) delaySource(ctx context.Context, domain string, wait time.Duration) error {
	return b.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		allowed := time.Now().UTC().Add(wait)
		if lease, err := tx.Traffic.Get(ctx, domain); err == nil && lease.NextAllowedAt.After(allowed) {
			allowed = lease.NextAllowedAt
		}
		return tx.Traffic.Save(ctx, domain, Lease{NextAllowedAt: allowed})
	})
}

func (b *SourceStore) reserveTraffic(ctx context.Context, domain string) (time.Time, error) {
	var allowed time.Time
	err := b.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		now := time.Now().UTC()
		allowed = now
		if lease, err := tx.Traffic.Get(ctx, domain); err == nil && lease.NextAllowedAt.After(now) {
			allowed = lease.NextAllowedAt
		}
		return tx.Traffic.Save(ctx, domain, Lease{NextAllowedAt: allowed.Add(b.pace())})
	})
	return allowed, err
}

func updateSource(health *model.SourceHealth, latency time.Duration, sourceErr error) {
	now := time.Now().UTC()
	health.LatencyMilliseconds = latency.Milliseconds()
	if sourceErr == nil {
		health.LastSuccess, health.ConsecutiveFailures, health.CircuitOpenUntil = &now, 0, nil
		return
	}
	health.LastFailure = &now
	// Un Freno Queda Anotado y no Avanza el Contador: cinco 429 seguidos son
	// una Tienda sana Pidiendo Calma, y Abrirle el Circuito la Deja muda un
	// Minuto por Portarse bien.
	if source.Throttled(sourceErr) {
		return
	}
	health.ConsecutiveFailures++
	if health.ConsecutiveFailures >= circuitThreshold {
		opened := now.Add(circuitWait(health.ConsecutiveFailures))
		health.CircuitOpenUntil = &opened
	}
}

// circuitWait Says how long a Fallen Store Stays unasked, Doubling with each
// further Fall.
//
// Un Minuto fijo Servia para una Caida de un rato y Fallaba para una de una
// Semana: `v3.netdecker.cl` Llevaba 39 Fallas seguidas y ocho Dias sin
// Contestar, y aun asi Recibia cinco Consultas por Minuto. Cada una le Costaba
// su Timeout a la Busqueda que la Pedia, y a la Tienda caida un Golpe mas.
//
// Doblar Respeta las dos Puntas: una Caida corta se Perdona rapido, y una
// larga se Consulta una vez por Hora hasta que Vuelva. El Tope Existe porque
// una Tienda que Vuelve no Deberia Esperar mas que eso para que la Noten.
func circuitWait(failures int) time.Duration {
	wait := circuitCooldown << min(failures-circuitThreshold, 16)
	return min(wait, circuitCeiling)
}

func waitUntil(ctx context.Context, target time.Time) error {
	wait := time.Until(target)
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
