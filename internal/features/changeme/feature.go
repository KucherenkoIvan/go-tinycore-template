// Package changeme is the placeholder feature — the layout to copy for real
// features, then delete. It has no transport adapters yet: add
// adapters/rest/ (or grpc/) when the first endpoint appears, and register
// routes from cmd/app/main.go.
package changeme

import (
	"github.com/KucherenkoIvan/go-kernel/ddd"
	"github.com/KucherenkoIvan/go-kernel/events"
	kernelsqlite "github.com/KucherenkoIvan/go-kernel/sqlite"

	sqliteadapter "github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/adapters/sqlite"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/application/ports"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/application/usecases/managechangeme"
)

// Feature is what the composition root (cmd/app/main.go) sees: the use-cases
// future transport adapters will call.
type Feature struct {
	Create *managechangeme.CreateCommand
	Update *managechangeme.UpdateCommand
	Delete *managechangeme.DeleteCommand
}

// New is the feature's composition root: port → adapter, then use-cases.
// No business logic here — only wiring.
func New(db *kernelsqlite.Client, pub *events.ChannelPublisher) *Feature {
	// ports → adapters
	var repo ports.ChangeMeRepository = sqliteadapter.NewChangeMeRepository(db)
	var producer ports.ChangeMeEventProducer = pub // an outbox adapter would slot in here
	var txManager ddd.TxManager = db

	// async reactions go here when they appear:
	// pub.Subscribe(events.Handler{Name: "...", Events: []string{...}, Handle: ...})

	return &Feature{
		Create: managechangeme.NewCreateCommand(txManager, ddd.UUIDv7Generator{}, ddd.SystemClock{}, repo, producer),
		Update: managechangeme.NewUpdateCommand(txManager, repo, producer),
		Delete: managechangeme.NewDeleteCommand(txManager, repo, producer),
	}
}
