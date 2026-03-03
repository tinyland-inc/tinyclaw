//go:build !nochinese

package channels

import (
	"github.com/tinyland-inc/tinyclaw/pkg/bus"
	"github.com/tinyland-inc/tinyclaw/pkg/config"
	"github.com/tinyland-inc/tinyclaw/pkg/logger"
)

// initChineseChannels initializes Feishu, QQ, DingTalk, OneBot, WeCom, and
// WeCom App channels. Excluded from builds with -tags nochinese.
//
//nolint:gocognit // repetitive channel initialization, one block per channel kind
func initChineseChannels(cfg *config.Config, messageBus *bus.MessageBus, channels map[string]Channel) {
	if cfg.Channels.Feishu.Enabled {
		logger.DebugC("channels", "Attempting to initialize Feishu channel")
		feishu, err := NewFeishuChannel(cfg.Channels.Feishu, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Feishu channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["feishu"] = feishu
			logger.InfoC("channels", "Feishu channel enabled successfully")
		}
	}

	if cfg.Channels.QQ.Enabled {
		logger.DebugC("channels", "Attempting to initialize QQ channel")
		qq, err := NewQQChannel(cfg.Channels.QQ, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize QQ channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["qq"] = qq
			logger.InfoC("channels", "QQ channel enabled successfully")
		}
	}

	if cfg.Channels.DingTalk.Enabled && cfg.Channels.DingTalk.ClientID != "" {
		logger.DebugC("channels", "Attempting to initialize DingTalk channel")
		dingtalk, err := NewDingTalkChannel(cfg.Channels.DingTalk, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize DingTalk channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["dingtalk"] = dingtalk
			logger.InfoC("channels", "DingTalk channel enabled successfully")
		}
	}

	if cfg.Channels.OneBot.Enabled && cfg.Channels.OneBot.WSUrl != "" {
		logger.DebugC("channels", "Attempting to initialize OneBot channel")
		onebot, err := NewOneBotChannel(cfg.Channels.OneBot, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize OneBot channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["onebot"] = onebot
			logger.InfoC("channels", "OneBot channel enabled successfully")
		}
	}

	if cfg.Channels.WeCom.Enabled && cfg.Channels.WeCom.Token != "" {
		logger.DebugC("channels", "Attempting to initialize WeCom channel")
		wecom, err := NewWeComBotChannel(cfg.Channels.WeCom, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize WeCom channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["wecom"] = wecom
			logger.InfoC("channels", "WeCom channel enabled successfully")
		}
	}

	if cfg.Channels.WeComApp.Enabled && cfg.Channels.WeComApp.CorpID != "" {
		logger.DebugC("channels", "Attempting to initialize WeCom App channel")
		wecomApp, err := NewWeComAppChannel(cfg.Channels.WeComApp, messageBus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize WeCom App channel", map[string]any{
				"error": err.Error(),
			})
		} else {
			channels["wecom_app"] = wecomApp
			logger.InfoC("channels", "WeCom App channel enabled successfully")
		}
	}
}
