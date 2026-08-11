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
	"time"

	"github.com/samber/do"
	"github.com/samber/oops"

	"hyperfocus/internal/client/twitch"
	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	httpHandlers "hyperfocus/internal/handlers/http"
	"hyperfocus/internal/migrations"
	"hyperfocus/internal/ocr"
	"hyperfocus/internal/pkg/clock"
	"hyperfocus/internal/pkg/steam"
	nottwitch "hyperfocus/internal/pkg/twitch"
	"hyperfocus/internal/previews"
	"hyperfocus/internal/repository/postgres"
	"hyperfocus/internal/server"
	"hyperfocus/internal/usecases/moments"
	"hyperfocus/internal/usecases/notify"
	"hyperfocus/internal/usecases/poll"
	"hyperfocus/internal/usecases/prune"
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

// Build constructs the application.
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
		return nil, oops.Wrapf(err, "failed to initialize Twitch client")
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
		Clock: clock.System(), Logger: log, Gateway: tw, Repo: repo,
		Preview: pv, OCR: ocrSvc, Config: cfg.Poll, OCRConfig: cfg.OCR,
		DataDir: cfg.Storage.DataDir,
	})
	momentsUC := moments.New(moments.Deps{Logger: log, Repo: repo})
	pruneUC := prune.New(prune.Deps{
		Clock: clock.System(), Logger: log, Repo: repo, Preview: pv, Config: cfg.Prune,
	})

	ircCommands := make(chan nottwitch.IRCCommand, 32)
	var subHandler *httpHandlers.SubscribeHandler

	bgCtx, cancel := context.WithCancel(ctx)
	bg := new(sync.WaitGroup)

	if cfg.Notify.IsEnabled() {
		botHelix, err := nottwitch.NewBotHelix(nottwitch.BotConfig{
			ClientID:     cfg.TwitchBot.ClientIDor(cfg.Twitch.ClientID),
			ClientSecret: cfg.TwitchBot.ClientSecretOr(cfg.Twitch.ClientSecret),
			RefreshToken: cfg.TwitchBot.RefreshToken,
		}, log)
		if err != nil {
			cancel()
			return nil, oops.Wrapf(err, "twitch bot init")
		}

		ircBot := nottwitch.NewIRCBot(log, cfg.TwitchBot.ClientIDor(cfg.Twitch.ClientID), botHelix.UserAccessToken, ircCommands)

		notifyUC := notify.New(notify.Deps{
			Logger:    log,
			Repo:      repo,
			IRC:       ircBot,
			Steam:     newSteamNameResolver(cfg.Steam, log),
			BotUserID: botHelix.BotUserID(),
			Config:    cfg.Notify,
		})

		subHandler = httpHandlers.NewSubscribeHandler(log, repo, botHelix, ircBot, cfg.Notify, cfg.Steam.APIKey)

		pollUC.AfterCycle = func(pCtx context.Context, snapshotID int64, samples []entity.StreamSample) {
			notifyUC.ProcessSnapshot(pCtx, snapshotID, samples)
		}

		channels, err := repo.ActiveSubscriberChannels(ctx)
		if err != nil {
			log.Warn("container: failed to load initial subscriber channels", slog.Any("error", err))
		} else {
			ircBot.Join(channels...)
		}

		bg.Add(8)
		go func() { defer bg.Done(); pollUC.Run(bgCtx) }()
		go func() { defer bg.Done(); pruneUC.Run(bgCtx) }()
		go func() { defer bg.Done(); tw.RunRefreshLoop(bgCtx) }()
		go func() { defer bg.Done(); botHelix.RefreshLoop(bgCtx) }()
		go func() { defer bg.Done(); ircCommandLoop(bgCtx, ircCommands, subHandler) }()
		go func() { defer bg.Done(); ircBot.Run(bgCtx) }()
		go func() { defer bg.Done(); channelRefreshLoop(bgCtx, ircBot, repo) }()
		go func() { defer bg.Done(); cleanupLoop(bgCtx, repo, log) }()
	} else {
		bg.Add(3)
		go func() { defer bg.Done(); pollUC.Run(bgCtx) }()
		go func() { defer bg.Done(); pruneUC.Run(bgCtx) }()
		go func() { defer bg.Done(); tw.RunRefreshLoop(bgCtx) }()
	}

	srv := server.New(log, httpHandlers.Deps{
		Logger:    log,
		Moments:   momentsUC,
		Streamers: repo,
		Previews:  pv,
		StatsRepo: repo,
		Version:   Version,
		Subscribe: subHandler,
	}, server.Config{Addr: cfg.Service.HTTPAddr, Version: Version})

	return &App{
		Server: srv,
		inj:    inj,
		cancel: cancel,
		bg:     bg,
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

func ircCommandLoop(ctx context.Context, commands <-chan nottwitch.IRCCommand, h *httpHandlers.SubscribeHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-commands:
			h.HandleIRCCommand(ctx, cmd)
		}
	}
}

func channelRefreshLoop(ctx context.Context, irc *nottwitch.IRCBot, repo interface{ ActiveSubscriberChannels(context.Context) ([]string, error) }) {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			chs, err := repo.ActiveSubscriberChannels(ctx)
			if err == nil {
				irc.Join(chs...)
			}
		}
	}
}

func cleanupLoop(ctx context.Context, repo interface{ DeletePendingExpired(context.Context) (int64, error) }, log *slog.Logger) {
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := repo.DeletePendingExpired(ctx)
			if err == nil && n > 0 {
				log.Info("notify: cleared expired pending subscriptions", slog.Int64("count", n))
			}
		}
	}
}

func newSteamNameResolver(cfg config.Steam, log *slog.Logger) *steamNameResolver {
	if cfg.APIKey == "" {
		return &steamNameResolver{}
	}
	return &steamNameResolver{apiKey: cfg.APIKey, log: log}
}

type steamNameResolver struct {
	apiKey string
	log    *slog.Logger
}

func (r *steamNameResolver) GetPlayerSummaries(ctx context.Context, steamIDs []string) ([]string, error) {
	if r.apiKey == "" || len(steamIDs) == 0 {
		return nil, nil
	}
	summaries, err := steam.GetPlayerSummaries(ctx, r.apiKey, steamIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]string, len(summaries))
	for _, s := range summaries {
		byID[s.SteamID] = s.PersonaName
	}
	out := make([]string, len(steamIDs))
	for i, id := range steamIDs {
		out[i] = byID[id]
	}
	return out, nil
}
