package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ToDhallOptions controls JSON-to-Dhall config migration.
type ToDhallOptions struct {
	ConfigPath string // JSON config path (default: ~/.picoclaw/config.json)
	OutputPath string // Dhall output path (default: ~/.picoclaw/config.dhall)
	DryRun     bool
	Force      bool
}

// ToDhallResult summarizes the conversion.
type ToDhallResult struct {
	OutputPath string
	Warnings   []string
}

// RunToDhall converts a JSON config file to Dhall format.
func RunToDhall(opts ToDhallOptions) (*ToDhallResult, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolving home directory: %w", err)
		}
		configPath = filepath.Join(home, ".picoclaw", "config.json")
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = strings.TrimSuffix(configPath, ".json") + ".dhall"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	result := &ToDhallResult{OutputPath: outputPath}
	dhall := configToDhall(cfg, result)

	if opts.DryRun {
		fmt.Println("-- Generated Dhall config (dry-run)")
		fmt.Println(dhall)
		return result, nil
	}

	if !opts.Force {
		if _, err := os.Stat(outputPath); err == nil {
			return nil, fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, []byte(dhall), 0o600); err != nil {
		return nil, err
	}

	return result, nil
}

// configToDhall renders a Config as Dhall source text.
func configToDhall(cfg *config.Config, result *ToDhallResult) string {
	var b strings.Builder

	b.WriteString("-- PicoClaw configuration (generated from JSON)\n")
	b.WriteString("-- Edit this file to manage your configuration as typed Dhall.\n\n")
	b.WriteString("let Types = https://raw.githubusercontent.com/tinyland-inc/picoclaw/main/dhall/types/package.dhall\n")
	b.WriteString("let H = https://raw.githubusercontent.com/tinyland-inc/picoclaw/main/dhall/helpers.dhall\n\n")

	b.WriteString("let emptyStrings = [] : List Text\n\n")

	// Render model_list entries
	if len(cfg.ModelList) > 0 {
		for i, mc := range cfg.ModelList {
			if mc.APIKey != "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("model_list[%d] (%s): credential value redacted", i, mc.ModelName))
			}
		}
	}

	b.WriteString("in  ")
	renderConfig(&b, cfg, "    ")

	return b.String()
}

func renderConfig(b *strings.Builder, cfg *config.Config, indent string) {
	b.WriteString("{ agents =\n")
	renderAgents(b, &cfg.Agents, indent)
	b.WriteString(fmt.Sprintf("%s, bindings = %s\n", indent, renderBindings(cfg.Bindings)))
	b.WriteString(fmt.Sprintf("%s, session =\n", indent))
	renderSession(b, &cfg.Session, indent+"  ")
	b.WriteString(fmt.Sprintf("%s, channels =\n", indent))
	renderChannels(b, &cfg.Channels, indent+"  ")
	b.WriteString(fmt.Sprintf("%s, model_list =\n", indent))
	renderModelList(b, cfg.ModelList, indent+"    ")
	b.WriteString(fmt.Sprintf("%s, gateway = { host = %s, port = %d }\n", indent,
		dhallText(cfg.Gateway.Host), cfg.Gateway.Port))
	b.WriteString(fmt.Sprintf("%s, tools =\n", indent))
	renderTools(b, &cfg.Tools, indent+"  ")
	b.WriteString(fmt.Sprintf("%s, heartbeat = { enabled = %s, interval = %d }\n", indent,
		dhallBool(cfg.Heartbeat.Enabled), cfg.Heartbeat.Interval))
	b.WriteString(fmt.Sprintf("%s, devices = { enabled = %s, monitor_usb = %s }\n", indent,
		dhallBool(cfg.Devices.Enabled), dhallBool(cfg.Devices.MonitorUSB)))
	b.WriteString(indent + "}\n")
}

func renderAgents(b *strings.Builder, agents *config.AgentsConfig, indent string) {
	b.WriteString(indent + "  { defaults =\n")
	d := agents.Defaults
	b.WriteString(indent + "    { workspace = " + dhallText(d.Workspace) + "\n")
	b.WriteString(indent + "    , restrict_to_workspace = " + dhallBool(d.RestrictToWorkspace) + "\n")
	b.WriteString(indent + "    , provider = " + dhallText(d.Provider) + "\n")
	b.WriteString(indent + "    , model_name = " + dhallOptionalText(d.ModelName) + "\n")
	b.WriteString(indent + "    , model = " + dhallOptionalText(d.GetModelName()) + "\n")
	b.WriteString(indent + "    , model_fallbacks = " + dhallTextList(d.ModelFallbacks) + "\n")
	b.WriteString(indent + "    , image_model = " + dhallOptionalText(d.ImageModel) + "\n")
	b.WriteString(indent + "    , image_model_fallbacks = " + dhallTextList(d.ImageModelFallbacks) + "\n")
	b.WriteString(indent + "    , max_tokens = " + fmt.Sprintf("%d", d.MaxTokens) + "\n")
	if d.Temperature != nil {
		b.WriteString(indent + "    , temperature = Some " + fmt.Sprintf("%g", *d.Temperature) + "\n")
	} else {
		b.WriteString(indent + "    , temperature = None Double\n")
	}
	b.WriteString(indent + "    , max_tool_iterations = " + fmt.Sprintf("%d", d.MaxToolIterations) + "\n")
	b.WriteString(indent + "    }\n")

	if len(agents.List) > 0 {
		b.WriteString(indent + "  , list =\n")
		b.WriteString(indent + "    [ ")
		for i, a := range agents.List {
			if i > 0 {
				b.WriteString(indent + "    , ")
			}
			renderAgentConfig(b, &a, indent+"      ")
			b.WriteString("\n")
		}
		b.WriteString(indent + "    ]\n")
	} else {
		b.WriteString(indent + "  , list = [] : List Types.Agent.AgentConfig\n")
	}
	b.WriteString(indent + "  }\n")
}

func renderAgentConfig(b *strings.Builder, a *config.AgentConfig, indent string) {
	b.WriteString("{ id = " + dhallText(a.ID))
	if a.Default {
		b.WriteString(", default = True")
	}
	if a.Name != "" {
		b.WriteString(", name = Some " + dhallText(a.Name))
	}
	if a.Workspace != "" {
		b.WriteString(", workspace = Some " + dhallText(a.Workspace))
	}
	b.WriteString(" }")
}

func renderBindings(bindings []config.AgentBinding) string {
	if len(bindings) == 0 {
		return "[] : List Types.Binding.AgentBinding"
	}
	parts := []string{}
	for _, bind := range bindings {
		bindJSON, _ := json.Marshal(bind)
		parts = append(parts, fmt.Sprintf("-- TODO: convert binding: %s", string(bindJSON)))
	}
	return "[\n" + strings.Join(parts, "\n") + "\n    ]"
}

func renderSession(b *strings.Builder, s *config.SessionConfig, indent string) {
	b.WriteString(indent + "{ dm_scope = " + dhallText(s.DMScope) + "\n")
	if len(s.IdentityLinks) > 0 {
		b.WriteString(indent + ", identity_links =\n")
		b.WriteString(indent + "  [ ")
		first := true
		for k, v := range s.IdentityLinks {
			if !first {
				b.WriteString(indent + "  , ")
			}
			b.WriteString(fmt.Sprintf("{ mapKey = %s, mapValue = %s }", dhallText(k), dhallTextList(v)))
			first = false
		}
		b.WriteString("\n" + indent + "  ]\n")
	} else {
		b.WriteString(indent + ", identity_links = [] : List { mapKey : Text, mapValue : List Text }\n")
	}
	b.WriteString(indent + "}\n")
}

func renderChannels(b *strings.Builder, ch *config.ChannelsConfig, indent string) {
	b.WriteString(indent + "{ whatsapp =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.WhatsApp.Enabled) + "\n")
	b.WriteString(indent + "  , bridge_url = " + dhallText(ch.WhatsApp.BridgeURL) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.WhatsApp.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", telegram =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.Telegram.Enabled) + "\n")
	b.WriteString(indent + "  , token = " + dhallText(ch.Telegram.Token) + "\n")
	b.WriteString(indent + "  , proxy = " + dhallText(ch.Telegram.Proxy) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.Telegram.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", feishu =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.Feishu.Enabled) + "\n")
	b.WriteString(indent + "  , app_id = " + dhallText(ch.Feishu.AppID) + "\n")
	b.WriteString(indent + "  , app_secret{- -} = " + dhallText(ch.Feishu.AppSecret) + "\n")
	b.WriteString(indent + "  , encrypt_key = " + dhallText(ch.Feishu.EncryptKey) + "\n")
	b.WriteString(indent + "  , verification_token = " + dhallText(ch.Feishu.VerificationToken) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.Feishu.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", discord =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.Discord.Enabled) + "\n")
	b.WriteString(indent + "  , token = " + dhallText(ch.Discord.Token) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.Discord.AllowFrom) + "\n")
	b.WriteString(indent + "  , mention_only = " + dhallBool(ch.Discord.MentionOnly) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", maixcam =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.MaixCam.Enabled) + "\n")
	b.WriteString(indent + "  , host = " + dhallText(ch.MaixCam.Host) + "\n")
	b.WriteString(indent + "  , port = " + fmt.Sprintf("%d", ch.MaixCam.Port) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.MaixCam.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", qq =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.QQ.Enabled) + "\n")
	b.WriteString(indent + "  , app_id = " + dhallText(ch.QQ.AppID) + "\n")
	b.WriteString(indent + "  , app_secret{- -} = " + dhallText(ch.QQ.AppSecret) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.QQ.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", dingtalk =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.DingTalk.Enabled) + "\n")
	b.WriteString(indent + "  , client_id = " + dhallText(ch.DingTalk.ClientID) + "\n")
	b.WriteString(indent + "  , client_secret{- -} = " + dhallText(ch.DingTalk.ClientSecret) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.DingTalk.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", slack =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.Slack.Enabled) + "\n")
	b.WriteString(indent + "  , bot_token = " + dhallText(ch.Slack.BotToken) + "\n")
	b.WriteString(indent + "  , app_token = " + dhallText(ch.Slack.AppToken) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.Slack.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", line =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.LINE.Enabled) + "\n")
	b.WriteString(indent + "  , channel_secret{- -} = " + dhallText(ch.LINE.ChannelSecret) + "\n")
	b.WriteString(indent + "  , channel_access_token{- -} = " + dhallText(ch.LINE.ChannelAccessToken) + "\n")
	b.WriteString(indent + "  , webhook_host = " + dhallText(ch.LINE.WebhookHost) + "\n")
	b.WriteString(indent + "  , webhook_port = " + fmt.Sprintf("%d", ch.LINE.WebhookPort) + "\n")
	b.WriteString(indent + "  , webhook_path = " + dhallText(ch.LINE.WebhookPath) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.LINE.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", onebot =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.OneBot.Enabled) + "\n")
	b.WriteString(indent + "  , ws_url = " + dhallText(ch.OneBot.WSUrl) + "\n")
	b.WriteString(indent + "  , access_token{- -} = " + dhallText(ch.OneBot.AccessToken) + "\n")
	b.WriteString(indent + "  , reconnect_interval = " + fmt.Sprintf("%d", ch.OneBot.ReconnectInterval) + "\n")
	b.WriteString(indent + "  , group_trigger_prefix = " + dhallTextList(ch.OneBot.GroupTriggerPrefix) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.OneBot.AllowFrom) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", wecom =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.WeCom.Enabled) + "\n")
	b.WriteString(indent + "  , token = " + dhallText(ch.WeCom.Token) + "\n")
	b.WriteString(indent + "  , encoding_aes_key = " + dhallText(ch.WeCom.EncodingAESKey) + "\n")
	b.WriteString(indent + "  , webhook_url = " + dhallText(ch.WeCom.WebhookURL) + "\n")
	b.WriteString(indent + "  , webhook_host = " + dhallText(ch.WeCom.WebhookHost) + "\n")
	b.WriteString(indent + "  , webhook_port = " + fmt.Sprintf("%d", ch.WeCom.WebhookPort) + "\n")
	b.WriteString(indent + "  , webhook_path = " + dhallText(ch.WeCom.WebhookPath) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.WeCom.AllowFrom) + "\n")
	b.WriteString(indent + "  , reply_timeout = " + fmt.Sprintf("%d", ch.WeCom.ReplyTimeout) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", wecom_app =\n")
	b.WriteString(indent + "  { enabled = " + dhallBool(ch.WeComApp.Enabled) + "\n")
	b.WriteString(indent + "  , corp_id = " + dhallText(ch.WeComApp.CorpID) + "\n")
	b.WriteString(indent + "  , corp_secret{- -} = " + dhallText(ch.WeComApp.CorpSecret) + "\n")
	b.WriteString(indent + "  , agent_id = " + dhallInt(int(ch.WeComApp.AgentID)) + "\n")
	b.WriteString(indent + "  , token = " + dhallText(ch.WeComApp.Token) + "\n")
	b.WriteString(indent + "  , encoding_aes_key = " + dhallText(ch.WeComApp.EncodingAESKey) + "\n")
	b.WriteString(indent + "  , webhook_host = " + dhallText(ch.WeComApp.WebhookHost) + "\n")
	b.WriteString(indent + "  , webhook_port = " + fmt.Sprintf("%d", ch.WeComApp.WebhookPort) + "\n")
	b.WriteString(indent + "  , webhook_path = " + dhallText(ch.WeComApp.WebhookPath) + "\n")
	b.WriteString(indent + "  , allow_from = " + dhallTextList(ch.WeComApp.AllowFrom) + "\n")
	b.WriteString(indent + "  , reply_timeout = " + fmt.Sprintf("%d", ch.WeComApp.ReplyTimeout) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + "}\n")
}

func renderModelList(b *strings.Builder, models []config.ModelConfig, indent string) {
	if len(models) == 0 {
		b.WriteString(indent + "[] : List Types.Provider.ModelConfig\n")
		return
	}
	b.WriteString(indent[:len(indent)-2] + "[ ")
	for i, mc := range models {
		if i > 0 {
			b.WriteString(indent[:len(indent)-2] + ", ")
		}
		renderModelConfig(b, &mc, indent)
		b.WriteString("\n")
	}
	b.WriteString(indent[:len(indent)-2] + "]\n")
}

func renderModelConfig(b *strings.Builder, mc *config.ModelConfig, indent string) {
	// Use helpers for standard entries, redact credentials
	apiBase := "None Text"
	if mc.APIBase != "" {
		apiBase = fmt.Sprintf("Some %s", dhallText(mc.APIBase))
	}

	// Redact API keys in output
	redactedKey := ""
	if mc.APIKey != "" {
		redactedKey = "env:PICOCLAW_API_KEY as Text"
	}

	if mc.AuthMethod != "" || mc.ConnectMode != "" || mc.Workspace != "" {
		// Full record for special providers
		b.WriteString("{ model_name = " + dhallText(mc.ModelName) + "\n")
		b.WriteString(indent + ", model = " + dhallText(mc.Model) + "\n")
		b.WriteString(indent + ", api_base = " + apiBase + "\n")
		b.WriteString(indent + ", api_key{- -} = " + dhallText(redactedKey) + "\n")
		b.WriteString(indent + ", proxy = None Text\n")
		b.WriteString(indent + ", auth_method = " + dhallOptionalText(mc.AuthMethod) + "\n")
		b.WriteString(indent + ", connect_mode = " + dhallOptionalText(mc.ConnectMode) + "\n")
		b.WriteString(indent + ", workspace = " + dhallOptionalText(mc.Workspace) + "\n")
		b.WriteString(indent + ", rpm = None Natural\n")
		b.WriteString(indent + ", max_tokens_field = None Text\n")
		b.WriteString(indent + "}")
	} else if mc.APIKey != "" {
		// Use mkModelConfig helper with env var
		b.WriteString(fmt.Sprintf("H.mkModelConfig %s %s %s (%s)",
			dhallText(mc.ModelName), dhallText(mc.Model), apiBase, redactedKey))
	} else {
		// Use emptyModelConfig helper
		b.WriteString(fmt.Sprintf("H.emptyModelConfig %s %s %s",
			dhallText(mc.ModelName), dhallText(mc.Model), apiBase))
	}
}

func renderTools(b *strings.Builder, t *config.ToolsConfig, indent string) {
	b.WriteString(indent + "{ web =\n")
	b.WriteString(indent + "  { brave = H.emptyBrave\n")
	b.WriteString(indent + "  , tavily = H.emptyTavily\n")
	b.WriteString(indent + "  , duckduckgo = { enabled = " + dhallBool(t.Web.DuckDuckGo.Enabled) +
		", max_results = " + fmt.Sprintf("%d", t.Web.DuckDuckGo.MaxResults) + " }\n")
	b.WriteString(indent + "  , perplexity = H.emptyPerplexity\n")
	b.WriteString(indent + "  , proxy = " + dhallText(t.Web.Proxy) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", cron = { exec_timeout_minutes = " + fmt.Sprintf("%d", t.Cron.ExecTimeoutMinutes) + " }\n")

	b.WriteString(indent + ", exec =\n")
	b.WriteString(indent + "  { enable_deny_patterns = " + dhallBool(t.Exec.EnableDenyPatterns) + "\n")
	b.WriteString(indent + "  , custom_deny_patterns = " + dhallTextList(t.Exec.CustomDenyPatterns) + "\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + ", skills =\n")
	b.WriteString(indent + "  { registries =\n")
	b.WriteString(indent + "    { clawhub =\n")
	ch := t.Skills.Registries.ClawHub
	b.WriteString(indent + "      { enabled = " + dhallBool(ch.Enabled) + "\n")
	b.WriteString(indent + "      , base_url = " + dhallText(ch.BaseURL) + "\n")
	b.WriteString(indent + "      , auth_token{- -} = " + dhallText(ch.AuthToken) + "\n")
	b.WriteString(indent + "      , search_path = " + dhallText(ch.SearchPath) + "\n")
	b.WriteString(indent + "      , skills_path = " + dhallText(ch.SkillsPath) + "\n")
	b.WriteString(indent + "      , download_path = " + dhallText(ch.DownloadPath) + "\n")
	b.WriteString(indent + "      , timeout = " + fmt.Sprintf("%d", ch.Timeout) + "\n")
	b.WriteString(indent + "      , max_zip_size = " + fmt.Sprintf("%d", ch.MaxZipSize) + "\n")
	b.WriteString(indent + "      , max_response_size = " + fmt.Sprintf("%d", ch.MaxResponseSize) + "\n")
	b.WriteString(indent + "      }\n")
	b.WriteString(indent + "    }\n")
	b.WriteString(indent + "  , max_concurrent_searches = " + fmt.Sprintf("%d", t.Skills.MaxConcurrentSearches) + "\n")
	b.WriteString(indent + "  , search_cache = { max_size = " + fmt.Sprintf("%d", t.Skills.SearchCache.MaxSize) +
		", ttl_seconds = " + fmt.Sprintf("%d", t.Skills.SearchCache.TTLSeconds) + " }\n")
	b.WriteString(indent + "  }\n")

	b.WriteString(indent + "}\n")
}

// Dhall literal helpers

func dhallText(s string) string {
	// Escape backslashes and double quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return fmt.Sprintf("%q", s)
}

func dhallBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func dhallOptionalText(s string) string {
	if s == "" {
		return "None Text"
	}
	return "Some " + dhallText(s)
}

func dhallInt(n int) string {
	if n >= 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}

func dhallTextList(ss []string) string {
	if len(ss) == 0 {
		return "emptyStrings"
	}
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = dhallText(s)
	}
	return "[ " + strings.Join(parts, ", ") + " ]"
}
