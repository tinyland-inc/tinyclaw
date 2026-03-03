// TinyClaw - Ultra-lightweight personal AI agent
// Inspired by Sipeed PicoClaw, itself inspired by OpenClaw
// License: MIT
//
// Copyright (c) 2026 TinyClaw contributors

package channels

import (
	"context"
	"fmt"
	"sync"

	"github.com/tinyland-inc/tinyclaw/pkg/bus"
	"github.com/tinyland-inc/tinyclaw/pkg/config"
	"github.com/tinyland-inc/tinyclaw/pkg/constants"
	"github.com/tinyland-inc/tinyclaw/pkg/logger"
)

type Manager struct {
	channels     map[string]Channel
	bus          *bus.MessageBus
	config       *config.Config
	dispatchTask *asyncTask
	mu           sync.RWMutex
}

type asyncTask struct {
	cancel context.CancelFunc
}

func NewManager(cfg *config.Config, messageBus *bus.MessageBus) (*Manager, error) {
	m := &Manager{
		channels: make(map[string]Channel),
		bus:      messageBus,
		config:   cfg,
	}

	if err := m.initChannels(); err != nil {
		return nil, err
	}

	return m, nil
}

//nolint:gocognit,gocyclo // initializes all channel types; one branch per channel kind
func (m *Manager) initChannels() error { //nolint:unparam // error return kept for future use
	logger.InfoC("channels", "Initializing channel manager")

	if m.config.Channels.Telegram.Enabled && m.config.Channels.Telegram.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Telegram channel")
		telegram, err := NewTelegramChannel(m.config, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Telegram channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["telegram"] = telegram
			logger.InfoC("channels", "Telegram channel enabled successfully")
		}
	}

	if m.config.Channels.WhatsApp.Enabled && m.config.Channels.WhatsApp.BridgeURL != "" {
		logger.DebugC("channels", "Attempting to initialize WhatsApp channel")
		whatsapp, err := NewWhatsAppChannel(m.config.Channels.WhatsApp, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize WhatsApp channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["whatsapp"] = whatsapp
			logger.InfoC("channels", "WhatsApp channel enabled successfully")
		}
	}

	// Chinese-only channels: Feishu, QQ, DingTalk, OneBot, WeCom, WeComApp
	// Excluded from build with -tags nochinese
	initChineseChannels(m.config, m.bus, m.channels)

	if m.config.Channels.Discord.Enabled && m.config.Channels.Discord.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Discord channel")
		discord, err := NewDiscordChannel(m.config.Channels.Discord, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Discord channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["discord"] = discord
			logger.InfoC("channels", "Discord channel enabled successfully")
		}
	}

	if m.config.Channels.MaixCam.Enabled {
		logger.DebugC("channels", "Attempting to initialize MaixCam channel")
		maixcam, err := NewMaixCamChannel(m.config.Channels.MaixCam, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize MaixCam channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["maixcam"] = maixcam
			logger.InfoC("channels", "MaixCam channel enabled successfully")
		}
	}

	if m.config.Channels.Slack.Enabled && m.config.Channels.Slack.BotToken != "" {
		logger.DebugC("channels", "Attempting to initialize Slack channel")
		slackCh, err := NewSlackChannel(m.config.Channels.Slack, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Slack channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["slack"] = slackCh
			logger.InfoC("channels", "Slack channel enabled successfully")
		}
	}

	if m.config.Channels.LINE.Enabled && m.config.Channels.LINE.ChannelAccessToken != "" {
		logger.DebugC("channels", "Attempting to initialize LINE channel")
		line, err := NewLINEChannel(m.config.Channels.LINE, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize LINE channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["line"] = line
			logger.InfoC("channels", "LINE channel enabled successfully")
		}
	}

	if m.config.Channels.XMPP.Enabled && m.config.Channels.XMPP.JID != "" {
		logger.DebugC("channels", "Attempting to initialize XMPP channel")
		xmppCh, err := NewXMPPChannel(m.config.Channels.XMPP, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize XMPP channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			m.channels["xmpp"] = xmppCh
			logger.InfoC("channels", "XMPP channel enabled successfully")
		}
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
		return nil
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}

	go m.dispatchOutbound(dispatchCtx)

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]any{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels started")
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Stopping channel", map[string]any{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

func (m *Manager) dispatchOutbound(ctx context.Context) {
	logger.InfoC("channels", "Outbound dispatcher started")

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("channels", "Outbound dispatcher stopped")
			return
		default:
			msg, ok := m.bus.SubscribeOutbound(ctx)
			if !ok {
				continue
			}

			// Silently skip internal channels
			if constants.IsInternalChannel(msg.Channel) {
				continue
			}

			m.mu.RLock()
			channel, exists := m.channels[msg.Channel]
			m.mu.RUnlock()

			if !exists {
				logger.WarnCF("channels", "Unknown channel for outbound message", map[string]any{
					"channel": msg.Channel,
				})
				continue
			}

			if err := channel.Send(ctx, msg); err != nil {
				logger.ErrorCF("channels", "Error sending message to channel", map[string]any{
					"channel": msg.Channel,
					"error":   err.Error(),
				})
			}
		}
	}
}

func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

func (m *Manager) GetStatus() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]any)
	for name, channel := range m.channels {
		status[name] = map[string]any{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
}

func (m *Manager) UnregisterChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, name)
}

func (m *Manager) SendToChannel(ctx context.Context, channelName, chatID, content string) error {
	m.mu.RLock()
	channel, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	msg := bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: content,
	}

	return channel.Send(ctx, msg)
}
