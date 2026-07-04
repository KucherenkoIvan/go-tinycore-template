package changeme_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/KucherenkoIvan/go-kernel/ddd"
	"github.com/KucherenkoIvan/go-kernel/events"

	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme"
	sqliteadapter "github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/adapters/sqlite"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/domain"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/shared/infra/storage"
)

// The CRUD flow over real components: in-memory sqlite with real migrations,
// the real channel publisher — no mocks. This is the test shape to keep for
// real features.
func TestCRUDFlow(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	db, err := storage.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pub := events.NewChannelPublisher(events.WithLogger(discard))

	// a test subscriber recording delivered events (registered before the
	// feature so ordering mirrors production wiring)
	var mu sync.Mutex
	var seen []string
	pub.Subscribe(events.Handler{
		Name: "test-recorder",
		Handle: func(_ context.Context, e ddd.DomainEvent) error {
			mu.Lock()
			defer mu.Unlock()
			seen = append(seen, e.EventName())
			return nil
		},
	})

	feature := changeme.New(db, pub)
	repo := sqliteadapter.NewChangeMeRepository(db) // direct adapter access for state asserts

	// create
	id, err := feature.Create.Execute(ctx, "first")
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err := repo.GetByID(ctx, ddd.NoTransaction, id)
	if err != nil || aggregate == nil || aggregate.Snapshot().Name != "first" {
		t.Fatalf("after create: %+v err=%v", aggregate, err)
	}

	// update
	if err := feature.Update.Execute(ctx, id, "second"); err != nil {
		t.Fatal(err)
	}
	aggregate, _ = repo.GetByID(ctx, ddd.NoTransaction, id)
	if aggregate.Snapshot().Name != "second" {
		t.Fatalf("after update: %+v", aggregate.Snapshot())
	}

	// domain errors propagate with their codes
	if err := feature.Update.Execute(ctx, id, "  "); err == nil || !ddd.IsDomainError(err) {
		t.Fatalf("blank update: %v", err)
	}
	var notFound *domain.ChangeMeNotFoundError
	if err := feature.Update.Execute(ctx, "missing", "x"); !errors.As(err, &notFound) {
		t.Fatalf("missing update: %v", err)
	}

	// delete
	if err := feature.Delete.Execute(ctx, id); err != nil {
		t.Fatal(err)
	}
	if aggregate, _ = repo.GetByID(ctx, ddd.NoTransaction, id); aggregate != nil {
		t.Fatal("aggregate must be gone after delete")
	}
	if err := feature.Delete.Execute(ctx, id); !errors.As(err, &notFound) {
		t.Fatalf("double delete: %v", err)
	}

	// commit-gated events were buffered on transactions; Close drains them
	// synchronously — the create/update/delete facts, in order.
	if err := pub.Close(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{domain.ChangeMeCreatedEventName, domain.ChangeMeUpdatedEventName, domain.ChangeMeDeletedEventName}
	if len(seen) != len(want) {
		t.Fatalf("events: got %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("events: got %v, want %v", seen, want)
		}
	}
}

// Failed transactions must publish nothing (commit-gating).
func TestNoGhostEvents(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	db, err := storage.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pub := events.NewChannelPublisher(events.WithLogger(discard))
	var count int
	var mu sync.Mutex
	pub.Subscribe(events.Handler{Name: "counter", Handle: func(context.Context, ddd.DomainEvent) error {
		mu.Lock()
		defer mu.Unlock()
		count++
		return nil
	}})

	feature := changeme.New(db, pub)

	if _, err := feature.Create.Execute(ctx, "keeper"); err != nil {
		t.Fatal(err)
	}
	_ = feature.Update.Execute(ctx, "missing", "x") // fails inside the tx

	_ = pub.Close(ctx)
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("delivered %d events, want 1 (failed tx must publish nothing)", count)
	}
}
