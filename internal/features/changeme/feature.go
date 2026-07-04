// Package changeme is the placeholder feature — the layout to copy for real
// features, then delete: aggregate + CRUD events, commands and queries,
// sqlite adapters, REST and gRPC transports.
package changeme

import (
	"github.com/KucherenkoIvan/go-kernel/ddd"
	"github.com/KucherenkoIvan/go-kernel/events"
	kernelsqlite "github.com/KucherenkoIvan/go-kernel/sqlite"

	grpcadapter "github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/adapters/grpc"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/adapters/rest"
	sqliteadapter "github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/adapters/sqlite"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/application/ports"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme/application/usecases/managechangeme"
)

// Feature is what the composition root (cmd/app/main.go) sees: the transport
// adapters to mount.
type Feature struct {
	Handlers *rest.Handlers                  // r.Group("/api") → Handlers.RegisterRoutes
	GRPC     *grpcadapter.ChangeMeController // changemev1.RegisterChangeMeServiceServer(srv, GRPC)
}

// New is the feature's composition root: port → adapter, then use-cases,
// then transports. No business logic here — only wiring.
func New(db *kernelsqlite.Client, pub *events.ChannelPublisher) *Feature {
	// ports → adapters
	var repo ports.ChangeMeRepository = sqliteadapter.NewChangeMeRepository(db)
	var reader ports.ChangeMeReader = sqliteadapter.NewChangeMeReader(db)
	var producer ports.ChangeMeEventProducer = pub // an outbox adapter would slot in here
	var txManager ddd.TxManager = db

	// use-cases
	create := managechangeme.NewCreateCommand(txManager, ddd.UUIDv7Generator{}, ddd.SystemClock{}, repo, producer)
	update := managechangeme.NewUpdateCommand(txManager, repo, producer)
	del := managechangeme.NewDeleteCommand(txManager, repo, producer)
	get := managechangeme.NewGetQuery(reader)
	list := managechangeme.NewListQuery(reader)

	// async reactions go here when they appear:
	// pub.Subscribe(events.Handler{Name: "...", Events: []string{...}, Handle: ...})

	return &Feature{
		Handlers: rest.NewHandlers(create, update, del, get, list),
		GRPC:     grpcadapter.NewChangeMeController(create, update, del, get, list),
	}
}
