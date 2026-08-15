package twitch

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"golang.org/x/time/rate"
)

type IRCCommand struct {
	SenderLogin string
	Channel     string
	Command     string // !hyperfocussub | !hyperfocusunsub
}

type IRCBot struct {
	client   *twitch.Client
	log      *slog.Logger
	nick     string
	tokenFn  func() string
	commands chan IRCCommand

	mu      sync.Mutex
	joined  map[string]bool
	limiter *rate.Limiter
}

func NewIRCBot(log *slog.Logger, nick string, tokenFn func() string, commands chan IRCCommand) *IRCBot {
	return &IRCBot{
		log:      log,
		nick:     nick,
		tokenFn:  tokenFn,
		commands: commands,
		joined:   make(map[string]bool),
		// Global rate limit for outbound messages: at most one per second.
		limiter: rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

func (b *IRCBot) Run(ctx context.Context) {
	b.log.Info("irc: starting read loop")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.runOnce(ctx)
	}
}

func (b *IRCBot) runOnce(ctx context.Context) {
	oauth := "oauth:" + b.tokenFn()
	b.mu.Lock()
	c := twitch.NewClient(b.nick, oauth)
	c.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		b.handleMessage(msg)
	})
	b.client = c
	chs := make([]string, 0, len(b.joined))
	for ch := range b.joined {
		if b.joined[ch] {
			chs = append(chs, ch)
		}
	}
	b.mu.Unlock()

	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(2 * time.Second):
		case <-done:
			return
		}
		b.mu.Lock()
		if b.client == c {
			for _, ch := range chs {
				c.Join(ch)
			}
		}
		b.mu.Unlock()
	}()

	b.log.Info("irc: connecting")
	if err := c.Connect(); err != nil {
		b.log.Warn("irc: connection ended", slog.Any("error", err))
	}
	close(done)
	b.log.Info("irc: disconnected, will reconnect")

	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
	}
}

func (b *IRCBot) Join(channels ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.client == nil {
		for _, ch := range channels {
			ch = strings.ToLower(strings.TrimSpace(ch))
			b.joined[ch] = true
		}
		return
	}
	for _, ch := range channels {
		ch = strings.ToLower(strings.TrimSpace(ch))
		if b.joined[ch] {
			continue
		}
		b.client.Join(ch)
		b.joined[ch] = true
		b.log.Info("irc: joined", slog.String("channel", ch))
	}
}

func (b *IRCBot) Part(channel string) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	b.mu.Lock()
	if !b.joined[channel] {
		b.mu.Unlock()
		return
	}
	b.joined[channel] = false
	if b.client != nil {
		b.client.Depart(channel)
	}
	b.mu.Unlock()
	b.log.Info("irc: parted", slog.String("channel", channel))
}

// Send delivers a message to a channel. Outbound messages are globally rate
// limited to one per second; concurrent callers block until their turn, or
// until ctx is cancelled.
func (b *IRCBot) Send(ctx context.Context, channel, message string) {
	if err := b.limiter.Wait(ctx); err != nil {
		return
	}
	b.mu.Lock()
	c := b.client
	b.mu.Unlock()
	if c != nil {
		c.Say(channel, message)
	}
}

func (b *IRCBot) handleMessage(msg twitch.PrivateMessage) {
	sender := strings.ToLower(msg.User.Name)
	channel := strings.ToLower(msg.Channel)

	if sender != channel {
		return
	}

	text := strings.TrimSpace(msg.Message)
	switch text {
	case "!hyperfocussub":
		select {
		case b.commands <- IRCCommand{SenderLogin: sender, Channel: channel, Command: "!hyperfocussub"}:
		default:
		}
	case "!hyperfocusunsub":
		select {
		case b.commands <- IRCCommand{SenderLogin: sender, Channel: channel, Command: "!hyperfocusunsub"}:
		default:
		}
	}
}
