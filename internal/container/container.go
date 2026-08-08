// Package container is the composition root: it wires configuration, the
// Postgres repository, the Twitch client, the preview store, the usecases, the
// background loops and the HTTP server. samber/do holds the infra providers;
// usecases are assembled with explicit dependency structs.
package container

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/samber/do"
	"github.com/samber/oops"

	"hyperfocus/internal/client/twitch"
	"hyperfocus/internal/config"
	httpHandlers "hyperfocus/internal/handlers/http"
	"hyperfocus/internal/migrations"
	"hyperfocus/internal/ocr"
	"hyperfocus/internal/pkg/clock"
	"hyperfocus/internal/previews"
	"hyperfocus/internal/repository/postgres"
	"hyperfocus/internal/server"
	"hyperfocus/internal/usecases/moments"
	"hyperfocus/internal/usecases/poll"
	"hyperfocus/internal/usecases/prune"
	"hyperfocus/internal/usecases/resolvevod"
)

// Version is injected at build time via ldflags (see Makefile).
var Version = "dev"

// App is the fully wired application.
type App struct {
	Server *http.Server
	inj    *do.Injector
	cancel context.CancelFunc
	bg     *sync.WaitGroup
	repo   *postgres.Repository
}

// Build constructs the application: infra providers are registered in the DI
// container, migrations are applied, usecases are assembled and background
// loops are started under a derived context.
func Build(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	inj := do.New()
	do.ProvideValue(inj, ctx)
	do.ProvideValue(inj, cfg)
	do.ProvideValue(inj, log)

	do.Provide(inj, func(i *do.Injector) (*postgres.Repository, error) {
		c := do.MustInvoke[*config.Config](i)
		ictx := do.MustInvoke[context.Context](i)
		return postgres.New(ictx, c.DB)
	})
	do.Provide(inj, func(i *do.Injector) (*twitch.Client, error) {
		c := do.MustInvoke[*config.Config](i)
		l := do.MustInvoke[*slog.Logger](i)
		return twitch.New(c.Twitch, l)
	})
	do.Provide(inj, func(i *do.Injector) (*previews.Store, error) {
		c := do.MustInvoke[*config.Config](i)
		return previews.New(c.Storage.DataDir)
	})

	repo, err := do.Invoke[*postgres.Repository](inj)
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if err := migrations.Run(ctx, repo.Pool()); err != nil {
		return nil, oops.Wrap(err)
	}
	log.Info("database migrations applied")

	tw, err := do.Invoke[*twitch.Client](inj)
	if err != nil {
		return nil, oops.Wrapf(err, "failed to initialize Twitch client (check twitch.client_id / twitch.client_secret)")
	}
	pv, err := do.Invoke[*previews.Store](inj)
	if err != nil {
		return nil, oops.Wrap(err)
	}

	ocrSvc, err := ocr.New(cfg.OCR, log)
	if err != nil {
		return nil, oops.Wrap(err)
	}

	pollUC := poll.New(poll.Deps{
		Clock: clock.System(), Logger: log, Gateway: tw, Vods: tw, Repo: repo,
		Preview: pv, OCR: ocrSvc, Config: cfg.Poll, OCRConfig: cfg.OCR,
	})
	resolveUC := resolvevod.New(resolvevod.Deps{
		Clock: clock.System(), Logger: log, Gateway: tw, Repo: repo, Config: cfg.Vod,
	})
	momentsUC := moments.New(moments.Deps{Logger: log, Repo: repo})
	pruneUC := prune.New(prune.Deps{
		Clock: clock.System(), Logger: log, Repo: repo, Preview: pv, Config: cfg.Prune,
	})

	bgCtx, cancel := context.WithCancel(ctx)
	var bg sync.WaitGroup
	bg.Add(4)
	go func() { defer bg.Done(); pollUC.Run(bgCtx) }()
	go func() { defer bg.Done(); resolveUC.Run(bgCtx) }()
	go func() { defer bg.Done(); pruneUC.Run(bgCtx) }()
	go func() { defer bg.Done(); tw.RunRefreshLoop(bgCtx) }()

	srv := server.New(log, httpHandlers.Deps{
		Logger:    log,
		Moments:   momentsUC,
		Streamers: repo,
		Vods:      repo,
		Previews:  pv,
		Version:   Version,
	}, server.Config{Addr: cfg.Service.HTTPAddr, Version: Version})

	return &App{
		Server: srv,
		inj:    inj,
		cancel: cancel,
		bg:     &bg,
		repo:   repo,
	}, nil
}

// Shutdown stops background loops gracefully and closes resources.
func (a *App) Shutdown() {
	a.cancel()
	a.bg.Wait()
	a.repo.Close()
	_ = a.inj.Shutdown()
}
