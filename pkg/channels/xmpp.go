package channels

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"

	"github.com/tinyland-inc/tinyclaw/pkg/bus"
	"github.com/tinyland-inc/tinyclaw/pkg/config"
	"github.com/tinyland-inc/tinyclaw/pkg/logger"
	"github.com/tinyland-inc/tinyclaw/pkg/utils"
)

const (
	xmppSendTimeout    = 10 * time.Second
	xmppMessageMaxLen  = 65536 // XMPP has no strict limit, but chunk large messages
	xmppReconnectDelay = 5 * time.Second
)

// xmppMessageBody is an XMPP message stanza with a body child element.
type xmppMessageBody struct {
	stanza.Message

	Body string `xml:"body"`
}

// XMPPChannel implements the Channel interface for XMPP with optional MUC support.
type XMPPChannel struct {
	*BaseChannel

	config  config.XMPPConfig
	session *xmpp.Session
	myJID   jid.JID
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
}

func NewXMPPChannel(cfg config.XMPPConfig, messageBus *bus.MessageBus) (*XMPPChannel, error) {
	if cfg.JID == "" {
		return nil, errors.New("xmpp jid is required")
	}
	if cfg.Password == "" {
		return nil, errors.New("xmpp password is required")
	}

	parsedJID, err := jid.Parse(cfg.JID)
	if err != nil {
		return nil, fmt.Errorf("invalid xmpp jid %q: %w", cfg.JID, err)
	}

	base := NewBaseChannel("xmpp", cfg, messageBus, cfg.AllowFrom)

	return &XMPPChannel{
		BaseChannel: base,
		config:      cfg,
		myJID:       parsedJID,
	}, nil
}

func (c *XMPPChannel) Start(ctx context.Context) error {
	logger.InfoC("xmpp", "Starting XMPP channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.connectWithCtx(c.ctx); err != nil {
		return fmt.Errorf("xmpp connection failed: %w", err)
	}

	c.setRunning(true)

	logger.InfoCF("xmpp", "XMPP channel connected", map[string]any{
		"jid":    c.myJID.String(),
		"server": c.serverAddr(),
	})

	go c.receiveLoop()

	return nil
}

func (c *XMPPChannel) Stop(ctx context.Context) error {
	logger.InfoC("xmpp", "Stopping XMPP channel")

	c.setRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	session := c.session
	c.session = nil
	c.mu.Unlock()

	if session != nil {
		if err := session.Close(); err != nil {
			logger.DebugCF("xmpp", "Error closing XMPP session", map[string]any{
				"error": err.Error(),
			})
		}
	}

	logger.InfoC("xmpp", "XMPP channel stopped")
	return nil
}

func (c *XMPPChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return errors.New("xmpp channel not running")
	}

	toJID, err := jid.Parse(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid recipient jid %q: %w", msg.ChatID, err)
	}

	if len(msg.Content) == 0 {
		return nil
	}

	chunks := utils.SplitMessage(msg.Content, xmppMessageMaxLen)
	for _, chunk := range chunks {
		if err := c.sendMessage(ctx, toJID, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (c *XMPPChannel) connectWithCtx(ctx context.Context) error {
	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	tlsConfig := &tls.Config{
		ServerName: c.myJID.Domain().String(),
		MinVersion: tls.VersionTLS12,
	}

	features := []xmpp.StreamFeature{
		xmpp.StartTLS(tlsConfig),
		xmpp.SASL("", c.config.Password, sasl.Plain, sasl.ScramSha1Plus, sasl.ScramSha1),
		xmpp.BindResource(),
	}

	var session *xmpp.Session
	var err error

	if addr := c.serverAddr(); addr != "" {
		// Custom server address: dial TCP then establish session
		conn, dialErr := net.DialTimeout("tcp", addr, 30*time.Second)
		if dialErr != nil {
			return fmt.Errorf("xmpp dial failed to %s: %w", addr, dialErr)
		}
		session, err = xmpp.NewClientSession(dialCtx, c.myJID, conn, features...)
	} else {
		// Default: SRV lookup via DialClientSession
		session, err = xmpp.DialClientSession(dialCtx, c.myJID, features...)
	}

	if err != nil {
		return fmt.Errorf("xmpp session negotiation failed: %w", err)
	}

	c.mu.Lock()
	c.session = session
	c.mu.Unlock()

	// Send initial presence to signal availability
	presenceErr := session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if presenceErr != nil {
		logger.WarnCF("xmpp", "Failed to send initial presence", map[string]any{
			"error": presenceErr.Error(),
		})
	}

	// Join MUC rooms if configured
	c.joinMUCRooms()

	return nil
}

//nolint:gocognit // receive loop handles XML stream parsing with multiple checks
func (c *XMPPChannel) receiveLoop() {
	logger.InfoC("xmpp", "XMPP receive loop started")

	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		logger.ErrorC("xmpp", "No session available for receive loop")
		return
	}

	err := session.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		// Only handle message stanzas
		if start.Name.Local != "message" {
			return nil
		}

		d := xml.NewTokenDecoder(xmlstream.Inner(t))
		msg := xmppMessageBody{}
		decodeErr := d.DecodeElement(&msg, start)
		if decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
			logger.DebugCF("xmpp", "Failed to decode message", map[string]any{
				"error": decodeErr.Error(),
			})
			return nil
		}

		// Populate From/To from StartElement attributes if not set by decode
		c.populateMessageAttrs(&msg, start)

		if msg.Body == "" {
			return nil
		}

		// Only handle chat and groupchat messages
		if msg.Type != stanza.ChatMessage && msg.Type != stanza.GroupChatMessage {
			return nil
		}

		c.handleIncoming(msg)
		return nil
	}))

	if err != nil && c.IsRunning() {
		logger.ErrorCF("xmpp", "XMPP receive loop ended with error", map[string]any{
			"error": err.Error(),
		})
		go c.reconnectLoop()
	}

	logger.InfoC("xmpp", "XMPP receive loop stopped")
}

func (c *XMPPChannel) populateMessageAttrs(msg *xmppMessageBody, start *xml.StartElement) {
	emptyJID := jid.JID{}
	if !msg.From.Equal(emptyJID) && !msg.To.Equal(emptyJID) {
		return
	}
	parsed, parseErr := stanza.NewMessage(*start)
	if parseErr != nil {
		return
	}
	if msg.From.Equal(emptyJID) {
		msg.From = parsed.From
	}
	if msg.To.Equal(emptyJID) {
		msg.To = parsed.To
	}
	if msg.Type == "" {
		msg.Type = parsed.Type
	}
}

func (c *XMPPChannel) handleIncoming(msg xmppMessageBody) {
	fromJID := msg.From
	if fromJID.Equal(jid.JID{}) {
		return
	}

	senderID := fromJID.Bare().String()

	// Skip our own messages (MUC echo)
	if fromJID.Bare().Equal(c.myJID.Bare()) {
		return
	}

	if !c.IsAllowed(senderID) {
		logger.DebugCF("xmpp", "Message rejected by allowlist", map[string]any{
			"sender": senderID,
		})
		return
	}

	chatID := fromJID.Bare().String()

	peerKind := "direct"
	if msg.Type == stanza.GroupChatMessage {
		peerKind = "groupchat"
	}

	metadata := map[string]string{
		"jid":       fromJID.String(),
		"bare_jid":  fromJID.Bare().String(),
		"peer_kind": peerKind,
	}

	if res := fromJID.Resourcepart(); res != "" {
		metadata["resource"] = res
	}

	logger.DebugCF("xmpp", "Received message", map[string]any{
		"sender": senderID,
		"type":   string(msg.Type),
	})

	c.HandleMessage(senderID, chatID, msg.Body, nil, metadata)
}

func (c *XMPPChannel) sendMessage(ctx context.Context, to jid.JID, content string) error {
	sendCtx, cancel := context.WithTimeout(ctx, xmppSendTimeout)
	defer cancel()

	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return errors.New("xmpp session not available")
	}

	// Determine message type based on whether destination is a MUC room
	msgType := stanza.ChatMessage
	for _, room := range c.config.MUCRooms {
		roomJID, parseErr := jid.Parse(room)
		if parseErr == nil && to.Bare().Equal(roomJID.Bare()) {
			msgType = stanza.GroupChatMessage
			break
		}
	}

	// Build <body>content</body> as a token reader
	bodyStart := xml.StartElement{Name: xml.Name{Local: "body"}}
	bodyPayload := xmlstream.Wrap(
		xmlstream.Token(xml.CharData(content)),
		bodyStart,
	)

	// Wrap body in <message to="..." type="...">
	msg := stanza.Message{
		To:   to,
		From: c.myJID,
		Type: msgType,
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Send(sendCtx, msg.Wrap(bodyPayload))
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to send xmpp message: %w", err)
		}
		return nil
	case <-sendCtx.Done():
		return fmt.Errorf("xmpp send timeout: %w", sendCtx.Err())
	}
}

func (c *XMPPChannel) joinMUCRooms() {
	for _, room := range c.config.MUCRooms {
		roomJID, err := jid.Parse(room)
		if err != nil {
			logger.ErrorCF("xmpp", "Invalid MUC room JID", map[string]any{
				"room":  room,
				"error": err.Error(),
			})
			continue
		}

		nick := c.myJID.Localpart()
		if nick == "" {
			nick = "tinyclaw"
		}

		fullRoomJID, err := jid.New(roomJID.Localpart(), roomJID.Domain().String(), nick)
		if err != nil {
			logger.ErrorCF("xmpp", "Failed to construct room JID with nick", map[string]any{
				"room":  room,
				"error": err.Error(),
			})
			continue
		}

		presence := stanza.Presence{
			To:   fullRoomJID,
			From: c.myJID,
			Type: stanza.AvailablePresence,
		}

		c.mu.Lock()
		session := c.session
		c.mu.Unlock()

		if session == nil {
			continue
		}

		if err := session.Send(c.ctx, presence.Wrap(nil)); err != nil {
			logger.ErrorCF("xmpp", "Failed to join MUC room", map[string]any{
				"room":  room,
				"error": err.Error(),
			})
		} else {
			logger.InfoCF("xmpp", "Joined MUC room", map[string]any{
				"room": room,
				"nick": nick,
			})
		}
	}
}

func (c *XMPPChannel) reconnectLoop() {
	for c.IsRunning() {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(xmppReconnectDelay):
		}

		logger.InfoC("xmpp", "Attempting XMPP reconnect")

		if err := c.connectWithCtx(c.ctx); err != nil {
			logger.ErrorCF("xmpp", "XMPP reconnect failed", map[string]any{
				"error": err.Error(),
			})
			continue
		}

		logger.InfoC("xmpp", "XMPP reconnected successfully")
		go c.receiveLoop()
		return
	}
}

func (c *XMPPChannel) serverAddr() string {
	if c.config.Server != "" {
		addr := c.config.Server
		if !strings.Contains(addr, ":") {
			addr += ":5222"
		}
		return addr
	}
	return ""
}
