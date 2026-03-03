package channels

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tinyland-inc/tinyclaw/pkg/bus"
	"github.com/tinyland-inc/tinyclaw/pkg/config"
)

func TestNewXMPPChannel(t *testing.T) {
	messageBus := bus.NewMessageBus()

	t.Run("valid config", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
			Server:   "xmpp.example.com:5222",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.NotNil(t, ch)
		assert.Equal(t, "xmpp", ch.Name())
		assert.False(t, ch.IsRunning())
	})

	t.Run("missing JID", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			Password: "secret",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.Error(t, err)
		assert.Nil(t, ch)
		assert.Contains(t, err.Error(), "jid is required")
	})

	t.Run("missing password", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled: true,
			JID:     "bot@example.com",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.Error(t, err)
		assert.Nil(t, ch)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("invalid JID", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "not a valid jid @@@",
			Password: "secret",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.Error(t, err)
		assert.Nil(t, ch)
		assert.Contains(t, err.Error(), "invalid xmpp jid")
	})

	t.Run("with allowlist", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:   true,
			JID:       "bot@example.com",
			Password:  "secret",
			AllowFrom: []string{"user@example.com", "admin@example.com"},
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.True(t, ch.IsAllowed("user@example.com"))
		assert.True(t, ch.IsAllowed("admin@example.com"))
		assert.False(t, ch.IsAllowed("stranger@example.com"))
	})

	t.Run("empty allowlist allows all", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.True(t, ch.IsAllowed("anyone@example.com"))
	})

	t.Run("implements Channel interface", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		var _ Channel = ch // compile-time check
	})
}

func TestXMPPChannelServerAddr(t *testing.T) {
	messageBus := bus.NewMessageBus()

	t.Run("custom server with port", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
			Server:   "xmpp.example.com:5222",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.Equal(t, "xmpp.example.com:5222", ch.serverAddr())
	})

	t.Run("custom server without port", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
			Server:   "xmpp.example.com",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.Equal(t, "xmpp.example.com:5222", ch.serverAddr())
	})

	t.Run("empty server for SRV lookup", func(t *testing.T) {
		cfg := config.XMPPConfig{
			Enabled:  true,
			JID:      "bot@example.com",
			Password: "secret",
		}
		ch, err := NewXMPPChannel(cfg, messageBus)
		require.NoError(t, err)
		assert.Empty(t, ch.serverAddr())
	})
}

func TestXMPPChannelSendNotRunning(t *testing.T) {
	messageBus := bus.NewMessageBus()
	cfg := config.XMPPConfig{
		Enabled:  true,
		JID:      "bot@example.com",
		Password: "secret",
	}
	ch, err := NewXMPPChannel(cfg, messageBus)
	require.NoError(t, err)

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "xmpp",
		ChatID:  "user@example.com",
		Content: "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestXMPPChannelSendEmptyContent(t *testing.T) {
	messageBus := bus.NewMessageBus()
	cfg := config.XMPPConfig{
		Enabled:  true,
		JID:      "bot@example.com",
		Password: "secret",
	}
	ch, err := NewXMPPChannel(cfg, messageBus)
	require.NoError(t, err)

	// Manually set running to test empty content path
	ch.setRunning(true)
	defer ch.setRunning(false)

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "xmpp",
		ChatID:  "user@example.com",
		Content: "",
	})
	assert.NoError(t, err) // empty content is a no-op
}

func TestXMPPChannelSendInvalidJID(t *testing.T) {
	messageBus := bus.NewMessageBus()
	cfg := config.XMPPConfig{
		Enabled:  true,
		JID:      "bot@example.com",
		Password: "secret",
	}
	ch, err := NewXMPPChannel(cfg, messageBus)
	require.NoError(t, err)

	ch.setRunning(true)
	defer ch.setRunning(false)

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "xmpp",
		ChatID:  "not valid @@@",
		Content: "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient jid")
}

func TestXMPPChannelMUCConfig(t *testing.T) {
	messageBus := bus.NewMessageBus()
	cfg := config.XMPPConfig{
		Enabled:  true,
		JID:      "bot@example.com",
		Password: "secret",
		MUCRooms: []string{"room1@conference.example.com", "room2@conference.example.com"},
	}
	ch, err := NewXMPPChannel(cfg, messageBus)
	require.NoError(t, err)
	assert.Len(t, ch.config.MUCRooms, 2)
}
