package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"github.com/caarlos0/env/v11"
)

// rrCounter is a global counter for round-robin load balancing across models.
var rrCounter atomic.Uint64

// ErrDhallNotAvailable is returned when dhall-to-json is not installed.
var ErrDhallNotAvailable = errors.New("dhall-to-json not available")

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

type Config struct {
	Agents    AgentsConfig    `json:"agents"`
	Bindings  []AgentBinding  `json:"bindings,omitempty"`
	Session   SessionConfig   `json:"session,omitzero"`
	Channels  ChannelsConfig  `json:"channels"`
	Providers ProvidersConfig `json:"providers,omitzero"`
	ModelList []ModelConfig   `json:"model_list"` // New model-centric provider configuration
	Gateway   GatewayConfig   `json:"gateway"`
	Tools     ToolsConfig     `json:"tools"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
	Devices   DevicesConfig   `json:"devices"`
	Tailscale TailscaleConfig `json:"tailscale,omitzero"`
	Aperture  ApertureConfig  `json:"aperture,omitzero"`
}

// MarshalJSON implements custom JSON marshaling for Config
// to omit providers section when empty and session when empty
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		*Alias

		Providers *ProvidersConfig `json:"providers,omitempty"`
		Session   *SessionConfig   `json:"session,omitempty"`
	}{
		Alias: (*Alias)(&c),
	}

	// Only include providers if not empty
	if !c.Providers.IsEmpty() {
		aux.Providers = &c.Providers
	}

	// Only include session if not empty
	if c.Session.DMScope != "" || len(c.Session.IdentityLinks) > 0 {
		aux.Session = &c.Session
	}

	return json.Marshal(aux)
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

// AgentModelConfig supports both string and structured model config.
// String format: "gpt-4" (just primary, no fallbacks)
// Object format: {"primary": "gpt-4", "fallbacks": ["claude-haiku"]}
type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		m.Fallbacks = nil
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

func (m AgentModelConfig) MarshalJSON() ([]byte, error) {
	if len(m.Fallbacks) == 0 && m.Primary != "" {
		return json.Marshal(m.Primary)
	}
	type raw struct {
		Primary   string   `json:"primary,omitempty"`
		Fallbacks []string `json:"fallbacks,omitempty"`
	}
	return json.Marshal(raw(m))
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type SessionConfig struct {
	DMScope       string              `json:"dm_scope,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
}

type AgentDefaults struct {
	Workspace           string   `env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"             json:"workspace"`
	RestrictToWorkspace bool     `env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE" json:"restrict_to_workspace"`
	Provider            string   `env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"              json:"provider"`
	ModelName           string   `env:"PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME"            json:"model_name,omitempty"`
	Model               string   `env:"PICOCLAW_AGENTS_DEFAULTS_MODEL"                 json:"model,omitempty"`                 // Deprecated: use model_name instead
	ModelFallbacks      []string `                                                     json:"model_fallbacks,omitempty"`       //nolint:tagalign // golines conflict
	ImageModel          string   `env:"PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL"           json:"image_model,omitempty"`           //nolint:tagalign // golines conflict
	ImageModelFallbacks []string `                                                     json:"image_model_fallbacks,omitempty"` //nolint:tagalign // golines conflict
	MaxTokens           int      `env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"            json:"max_tokens"`
	Temperature         *float64 `env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"           json:"temperature,omitempty"`
	MaxToolIterations   int      `env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"   json:"max_tool_iterations"`
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *AgentDefaults) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Feishu   FeishuConfig   `json:"feishu"`
	Discord  DiscordConfig  `json:"discord"`
	MaixCam  MaixCamConfig  `json:"maixcam"`
	QQ       QQConfig       `json:"qq"`
	DingTalk DingTalkConfig `json:"dingtalk"`
	Slack    SlackConfig    `json:"slack"`
	LINE     LINEConfig     `json:"line"`
	OneBot   OneBotConfig   `json:"onebot"`
	WeCom    WeComConfig    `json:"wecom"`
	WeComApp WeComAppConfig `json:"wecom_app"`
}

type WhatsAppConfig struct {
	Enabled   bool                `env:"PICOCLAW_CHANNELS_WHATSAPP_ENABLED"    json:"enabled"`
	BridgeURL string              `env:"PICOCLAW_CHANNELS_WHATSAPP_BRIDGE_URL" json:"bridge_url"`
	AllowFrom FlexibleStringSlice `env:"PICOCLAW_CHANNELS_WHATSAPP_ALLOW_FROM" json:"allow_from"`
}

type TelegramConfig struct {
	Enabled   bool                `env:"PICOCLAW_CHANNELS_TELEGRAM_ENABLED"    json:"enabled"`
	Token     string              `env:"PICOCLAW_CHANNELS_TELEGRAM_TOKEN"      json:"token"`
	Proxy     string              `env:"PICOCLAW_CHANNELS_TELEGRAM_PROXY"      json:"proxy"`
	AllowFrom FlexibleStringSlice `env:"PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM" json:"allow_from"`
}

type FeishuConfig struct {
	Enabled           bool                `env:"PICOCLAW_CHANNELS_FEISHU_ENABLED"            json:"enabled"`
	AppID             string              `env:"PICOCLAW_CHANNELS_FEISHU_APP_ID"             json:"app_id"`
	AppSecret         string              `env:"PICOCLAW_CHANNELS_FEISHU_APP_SECRET"         json:"app_secret"`
	EncryptKey        string              `env:"PICOCLAW_CHANNELS_FEISHU_ENCRYPT_KEY"        json:"encrypt_key"`
	VerificationToken string              `env:"PICOCLAW_CHANNELS_FEISHU_VERIFICATION_TOKEN" json:"verification_token"`
	AllowFrom         FlexibleStringSlice `env:"PICOCLAW_CHANNELS_FEISHU_ALLOW_FROM"         json:"allow_from"`
}

type DiscordConfig struct {
	Enabled     bool                `env:"PICOCLAW_CHANNELS_DISCORD_ENABLED"      json:"enabled"`
	Token       string              `env:"PICOCLAW_CHANNELS_DISCORD_TOKEN"        json:"token"`
	AllowFrom   FlexibleStringSlice `env:"PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM"   json:"allow_from"`
	MentionOnly bool                `env:"PICOCLAW_CHANNELS_DISCORD_MENTION_ONLY" json:"mention_only"`
}

type MaixCamConfig struct {
	Enabled   bool                `env:"PICOCLAW_CHANNELS_MAIXCAM_ENABLED"    json:"enabled"`
	Host      string              `env:"PICOCLAW_CHANNELS_MAIXCAM_HOST"       json:"host"`
	Port      int                 `env:"PICOCLAW_CHANNELS_MAIXCAM_PORT"       json:"port"`
	AllowFrom FlexibleStringSlice `env:"PICOCLAW_CHANNELS_MAIXCAM_ALLOW_FROM" json:"allow_from"`
}

type QQConfig struct {
	Enabled   bool                `env:"PICOCLAW_CHANNELS_QQ_ENABLED"    json:"enabled"`
	AppID     string              `env:"PICOCLAW_CHANNELS_QQ_APP_ID"     json:"app_id"`
	AppSecret string              `env:"PICOCLAW_CHANNELS_QQ_APP_SECRET" json:"app_secret"`
	AllowFrom FlexibleStringSlice `env:"PICOCLAW_CHANNELS_QQ_ALLOW_FROM" json:"allow_from"`
}

type DingTalkConfig struct {
	Enabled      bool                `env:"PICOCLAW_CHANNELS_DINGTALK_ENABLED"       json:"enabled"`
	ClientID     string              `env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_ID"     json:"client_id"`
	ClientSecret string              `env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_SECRET" json:"client_secret"`
	AllowFrom    FlexibleStringSlice `env:"PICOCLAW_CHANNELS_DINGTALK_ALLOW_FROM"    json:"allow_from"`
}

type SlackConfig struct {
	Enabled   bool                `env:"PICOCLAW_CHANNELS_SLACK_ENABLED"    json:"enabled"`
	BotToken  string              `env:"PICOCLAW_CHANNELS_SLACK_BOT_TOKEN"  json:"bot_token"`
	AppToken  string              `env:"PICOCLAW_CHANNELS_SLACK_APP_TOKEN"  json:"app_token"`
	AllowFrom FlexibleStringSlice `env:"PICOCLAW_CHANNELS_SLACK_ALLOW_FROM" json:"allow_from"`
}

type LINEConfig struct {
	Enabled            bool                `env:"PICOCLAW_CHANNELS_LINE_ENABLED"              json:"enabled"`
	ChannelSecret      string              `env:"PICOCLAW_CHANNELS_LINE_CHANNEL_SECRET"       json:"channel_secret"`
	ChannelAccessToken string              `env:"PICOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN" json:"channel_access_token"`
	WebhookHost        string              `env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_HOST"         json:"webhook_host"`
	WebhookPort        int                 `env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PORT"         json:"webhook_port"`
	WebhookPath        string              `env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PATH"         json:"webhook_path"`
	AllowFrom          FlexibleStringSlice `env:"PICOCLAW_CHANNELS_LINE_ALLOW_FROM"           json:"allow_from"`
}

type OneBotConfig struct {
	Enabled            bool                `env:"PICOCLAW_CHANNELS_ONEBOT_ENABLED"              json:"enabled"`
	WSUrl              string              `env:"PICOCLAW_CHANNELS_ONEBOT_WS_URL"               json:"ws_url"`
	AccessToken        string              `env:"PICOCLAW_CHANNELS_ONEBOT_ACCESS_TOKEN"         json:"access_token"`
	ReconnectInterval  int                 `env:"PICOCLAW_CHANNELS_ONEBOT_RECONNECT_INTERVAL"   json:"reconnect_interval"`
	GroupTriggerPrefix []string            `env:"PICOCLAW_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX" json:"group_trigger_prefix"`
	AllowFrom          FlexibleStringSlice `env:"PICOCLAW_CHANNELS_ONEBOT_ALLOW_FROM"           json:"allow_from"`
}

type WeComConfig struct {
	Enabled        bool                `env:"PICOCLAW_CHANNELS_WECOM_ENABLED"          json:"enabled"`
	Token          string              `env:"PICOCLAW_CHANNELS_WECOM_TOKEN"            json:"token"`
	EncodingAESKey string              `env:"PICOCLAW_CHANNELS_WECOM_ENCODING_AES_KEY" json:"encoding_aes_key"`
	WebhookURL     string              `env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_URL"      json:"webhook_url"`
	WebhookHost    string              `env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_HOST"     json:"webhook_host"`
	WebhookPort    int                 `env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_PORT"     json:"webhook_port"`
	WebhookPath    string              `env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_PATH"     json:"webhook_path"`
	AllowFrom      FlexibleStringSlice `env:"PICOCLAW_CHANNELS_WECOM_ALLOW_FROM"       json:"allow_from"`
	ReplyTimeout   int                 `env:"PICOCLAW_CHANNELS_WECOM_REPLY_TIMEOUT"    json:"reply_timeout"`
}

type WeComAppConfig struct {
	Enabled        bool                `env:"PICOCLAW_CHANNELS_WECOM_APP_ENABLED"          json:"enabled"`
	CorpID         string              `env:"PICOCLAW_CHANNELS_WECOM_APP_CORP_ID"          json:"corp_id"`
	CorpSecret     string              `env:"PICOCLAW_CHANNELS_WECOM_APP_CORP_SECRET"      json:"corp_secret"`
	AgentID        int64               `env:"PICOCLAW_CHANNELS_WECOM_APP_AGENT_ID"         json:"agent_id"`
	Token          string              `env:"PICOCLAW_CHANNELS_WECOM_APP_TOKEN"            json:"token"`
	EncodingAESKey string              `env:"PICOCLAW_CHANNELS_WECOM_APP_ENCODING_AES_KEY" json:"encoding_aes_key"`
	WebhookHost    string              `env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_HOST"     json:"webhook_host"`
	WebhookPort    int                 `env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_PORT"     json:"webhook_port"`
	WebhookPath    string              `env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_PATH"     json:"webhook_path"`
	AllowFrom      FlexibleStringSlice `env:"PICOCLAW_CHANNELS_WECOM_APP_ALLOW_FROM"       json:"allow_from"`
	ReplyTimeout   int                 `env:"PICOCLAW_CHANNELS_WECOM_APP_REPLY_TIMEOUT"    json:"reply_timeout"`
}

type HeartbeatConfig struct {
	Enabled  bool `env:"PICOCLAW_HEARTBEAT_ENABLED"  json:"enabled"`
	Interval int  `env:"PICOCLAW_HEARTBEAT_INTERVAL" json:"interval"` // minutes, min 5
}

type DevicesConfig struct {
	Enabled    bool `env:"PICOCLAW_DEVICES_ENABLED"     json:"enabled"`
	MonitorUSB bool `env:"PICOCLAW_DEVICES_MONITOR_USB" json:"monitor_usb"`
}

// TailscaleConfig holds Tailscale tsnet integration settings.
type TailscaleConfig struct {
	Enabled  bool   `env:"PICOCLAW_TAILSCALE_ENABLED"   json:"enabled"`
	Hostname string `env:"PICOCLAW_TAILSCALE_HOSTNAME"  json:"hostname"`
	StateDir string `env:"PICOCLAW_TAILSCALE_STATE_DIR" json:"state_dir"`
	AuthKey  string `env:"PICOCLAW_TAILSCALE_AUTH_KEY"  json:"auth_key"`
}

// ApertureConfig holds Tailscale Aperture proxy integration settings.
type ApertureConfig struct {
	Enabled    bool   `env:"PICOCLAW_APERTURE_ENABLED"     json:"enabled"`
	ProxyURL   string `env:"PICOCLAW_APERTURE_PROXY_URL"   json:"proxy_url"`
	WebhookURL string `env:"PICOCLAW_APERTURE_WEBHOOK_URL" json:"webhook_url"`
	WebhookKey string `env:"PICOCLAW_APERTURE_WEBHOOK_KEY" json:"webhook_key"`
	CerbosURL  string `env:"PICOCLAW_APERTURE_CERBOS_URL"  json:"cerbos_url"`
}

type ProvidersConfig struct {
	Anthropic     ProviderConfig       `json:"anthropic"`
	OpenAI        OpenAIProviderConfig `json:"openai"`
	OpenRouter    ProviderConfig       `json:"openrouter"`
	Groq          ProviderConfig       `json:"groq"`
	Zhipu         ProviderConfig       `json:"zhipu"`
	VLLM          ProviderConfig       `json:"vllm"`
	Gemini        ProviderConfig       `json:"gemini"`
	Nvidia        ProviderConfig       `json:"nvidia"`
	Ollama        ProviderConfig       `json:"ollama"`
	Moonshot      ProviderConfig       `json:"moonshot"`
	ShengSuanYun  ProviderConfig       `json:"shengsuanyun"`
	DeepSeek      ProviderConfig       `json:"deepseek"`
	Cerebras      ProviderConfig       `json:"cerebras"`
	VolcEngine    ProviderConfig       `json:"volcengine"`
	GitHubCopilot ProviderConfig       `json:"github_copilot"`
	Antigravity   ProviderConfig       `json:"antigravity"`
	Qwen          ProviderConfig       `json:"qwen"`
	Mistral       ProviderConfig       `json:"mistral"`
}

// IsEmpty checks if all provider configs are empty (no API keys or API bases set)
// Note: WebSearch is an optimization option and doesn't count as "non-empty"
//
//nolint:gocyclo // exhaustive provider field check; one condition per provider
func (p ProvidersConfig) IsEmpty() bool {
	return p.Anthropic.APIKey == "" && p.Anthropic.APIBase == "" &&
		p.OpenAI.APIKey == "" && p.OpenAI.APIBase == "" &&
		p.OpenRouter.APIKey == "" && p.OpenRouter.APIBase == "" &&
		p.Groq.APIKey == "" && p.Groq.APIBase == "" &&
		p.Zhipu.APIKey == "" && p.Zhipu.APIBase == "" &&
		p.VLLM.APIKey == "" && p.VLLM.APIBase == "" &&
		p.Gemini.APIKey == "" && p.Gemini.APIBase == "" &&
		p.Nvidia.APIKey == "" && p.Nvidia.APIBase == "" &&
		p.Ollama.APIKey == "" && p.Ollama.APIBase == "" &&
		p.Moonshot.APIKey == "" && p.Moonshot.APIBase == "" &&
		p.ShengSuanYun.APIKey == "" && p.ShengSuanYun.APIBase == "" &&
		p.DeepSeek.APIKey == "" && p.DeepSeek.APIBase == "" &&
		p.Cerebras.APIKey == "" && p.Cerebras.APIBase == "" &&
		p.VolcEngine.APIKey == "" && p.VolcEngine.APIBase == "" &&
		p.GitHubCopilot.APIKey == "" && p.GitHubCopilot.APIBase == "" &&
		p.Antigravity.APIKey == "" && p.Antigravity.APIBase == "" &&
		p.Qwen.APIKey == "" && p.Qwen.APIBase == "" &&
		p.Mistral.APIKey == "" && p.Mistral.APIBase == ""
}

// MarshalJSON implements custom JSON marshaling for ProvidersConfig
// to omit the entire section when empty
func (p ProvidersConfig) MarshalJSON() ([]byte, error) {
	if p.IsEmpty() {
		return []byte("null"), nil
	}
	type Alias ProvidersConfig
	return json.Marshal((*Alias)(&p))
}

type ProviderConfig struct {
	APIKey      string `env:"PICOCLAW_PROVIDERS_{{.Name}}_API_KEY"      json:"api_key"`
	APIBase     string `env:"PICOCLAW_PROVIDERS_{{.Name}}_API_BASE"     json:"api_base"`
	Proxy       string `env:"PICOCLAW_PROVIDERS_{{.Name}}_PROXY"        json:"proxy,omitempty"`
	AuthMethod  string `env:"PICOCLAW_PROVIDERS_{{.Name}}_AUTH_METHOD"  json:"auth_method,omitempty"`
	ConnectMode string `env:"PICOCLAW_PROVIDERS_{{.Name}}_CONNECT_MODE" json:"connect_mode,omitempty"` // only for Github Copilot, `stdio` or `grpc`
}

type OpenAIProviderConfig struct {
	ProviderConfig

	WebSearch bool `env:"PICOCLAW_PROVIDERS_OPENAI_WEB_SEARCH" json:"web_search"`
}

// ModelConfig represents a model-centric provider configuration.
// It allows adding new providers (especially OpenAI-compatible ones) via configuration only.
// The model field uses protocol prefix format: [protocol/]model-identifier
// Supported protocols: openai, anthropic, antigravity, claude-cli, codex-cli, github-copilot
// Default protocol is "openai" if no prefix is specified.
type ModelConfig struct {
	// Required fields
	ModelName string `json:"model_name"` // User-facing alias for the model
	Model     string `json:"model"`      // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")

	// HTTP-based providers
	APIBase string `json:"api_base,omitempty"` // API endpoint URL
	APIKey  string `json:"api_key"`            // API authentication key
	Proxy   string `json:"proxy,omitempty"`    // HTTP proxy URL

	// Special providers (CLI-based, OAuth, etc.)
	AuthMethod  string `json:"auth_method,omitempty"`  // Authentication method: oauth, token
	ConnectMode string `json:"connect_mode,omitempty"` // Connection mode: stdio, grpc
	Workspace   string `json:"workspace,omitempty"`    // Workspace path for CLI-based providers

	// Optional optimizations
	RPM            int    `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
}

// Validate checks if the ModelConfig has all required fields.
func (c *ModelConfig) Validate() error {
	if c.ModelName == "" {
		return errors.New("model_name is required")
	}
	if c.Model == "" {
		return errors.New("model is required")
	}
	return nil
}

type GatewayConfig struct {
	Host string `env:"PICOCLAW_GATEWAY_HOST" json:"host"`
	Port int    `env:"PICOCLAW_GATEWAY_PORT" json:"port"`
}

type BraveConfig struct {
	Enabled    bool   `env:"PICOCLAW_TOOLS_WEB_BRAVE_ENABLED"     json:"enabled"`
	APIKey     string `env:"PICOCLAW_TOOLS_WEB_BRAVE_API_KEY"     json:"api_key"`
	MaxResults int    `env:"PICOCLAW_TOOLS_WEB_BRAVE_MAX_RESULTS" json:"max_results"`
}

type TavilyConfig struct {
	Enabled    bool   `env:"PICOCLAW_TOOLS_WEB_TAVILY_ENABLED"     json:"enabled"`
	APIKey     string `env:"PICOCLAW_TOOLS_WEB_TAVILY_API_KEY"     json:"api_key"`
	BaseURL    string `env:"PICOCLAW_TOOLS_WEB_TAVILY_BASE_URL"    json:"base_url"`
	MaxResults int    `env:"PICOCLAW_TOOLS_WEB_TAVILY_MAX_RESULTS" json:"max_results"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `env:"PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED"     json:"enabled"`
	MaxResults int  `env:"PICOCLAW_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS" json:"max_results"`
}

type PerplexityConfig struct {
	Enabled    bool   `env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_ENABLED"     json:"enabled"`
	APIKey     string `env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_API_KEY"     json:"api_key"`
	MaxResults int    `env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_MAX_RESULTS" json:"max_results"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	Tavily     TavilyConfig     `json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
	Perplexity PerplexityConfig `json:"perplexity"`
	// Proxy is an optional proxy URL for web tools (http/https/socks5/socks5h).
	// For authenticated proxies, prefer HTTP_PROXY/HTTPS_PROXY env vars instead of embedding credentials in config.
	Proxy string `env:"PICOCLAW_TOOLS_WEB_PROXY" json:"proxy,omitempty"`
}

type CronToolsConfig struct {
	ExecTimeoutMinutes int `env:"PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES" json:"exec_timeout_minutes"` // 0 means no timeout
}

type ExecConfig struct {
	EnableDenyPatterns bool     `env:"PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS" json:"enable_deny_patterns"`
	CustomDenyPatterns []string `env:"PICOCLAW_TOOLS_EXEC_CUSTOM_DENY_PATTERNS" json:"custom_deny_patterns"`
}

type ToolsConfig struct {
	Web    WebToolsConfig    `json:"web"`
	Cron   CronToolsConfig   `json:"cron"`
	Exec   ExecConfig        `json:"exec"`
	Skills SkillsToolsConfig `json:"skills"`
}

type SkillsToolsConfig struct {
	Registries            SkillsRegistriesConfig `json:"registries"`
	MaxConcurrentSearches int                    `json:"max_concurrent_searches" env:"PICOCLAW_SKILLS_MAX_CONCURRENT_SEARCHES"` //nolint:tagalign // golines conflict
	SearchCache           SearchCacheConfig      `json:"search_cache"`
}

type SearchCacheConfig struct {
	MaxSize    int `env:"PICOCLAW_SKILLS_SEARCH_CACHE_MAX_SIZE"    json:"max_size"`
	TTLSeconds int `env:"PICOCLAW_SKILLS_SEARCH_CACHE_TTL_SECONDS" json:"ttl_seconds"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_ENABLED"           json:"enabled"`
	BaseURL         string `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"          json:"base_url"`
	AuthToken       string `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"        json:"auth_token"`
	SearchPath      string `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"       json:"search_path"`
	SkillsPath      string `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"       json:"skills_path"`
	DownloadPath    string `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"     json:"download_path"`
	Timeout         int    `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"           json:"timeout"`
	MaxZipSize      int    `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"      json:"max_zip_size"`
	MaxResponseSize int    `env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE" json:"max_response_size"`
}

// LoadDhallConfig loads configuration from a .dhall file by invoking dhall-to-json
// and parsing the resulting JSON. Returns nil, nil if dhall-to-json is not available.
func LoadDhallConfig(path string) (*Config, error) {
	dhallBin, err := exec.LookPath("dhall-to-json")
	if errors.Is(err, exec.ErrNotFound) {
		return nil, ErrDhallNotAvailable
	}
	if err != nil {
		return nil, fmt.Errorf("dhall-to-json lookup: %w", err)
	}

	cmd := exec.Command(dhallBin, "--file", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dhall-to-json failed for %s: %w\n%s", path, err, stderr.String())
	}

	cfg := DefaultConfig()

	// Same pre-scan logic as LoadConfig to avoid inheriting default model_list
	var tmp Config
	if err := json.Unmarshal(out, &tmp); err != nil {
		return nil, fmt.Errorf("error parsing dhall-to-json output: %w", err)
	}
	if len(tmp.ModelList) > 0 {
		cfg.ModelList = nil
	}

	if err := json.Unmarshal(out, cfg); err != nil {
		return nil, fmt.Errorf("error parsing dhall-to-json output: %w", err)
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if len(cfg.ModelList) == 0 && cfg.HasProvidersConfig() {
		cfg.ModelList = ConvertProvidersToModelList(cfg)
	}

	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	// Pre-scan the JSON to check how many model_list entries the user provided.
	// Go's JSON decoder reuses existing slice backing-array elements rather than
	// zero-initializing them, so fields absent from the user's JSON (e.g. api_base)
	// would silently inherit values from the DefaultConfig template at the same
	// index position. We only reset cfg.ModelList when the user actually provides
	// entries; when count is 0 we keep DefaultConfig's built-in list as fallback.
	var tmp Config
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	if len(tmp.ModelList) > 0 {
		cfg.ModelList = nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Auto-migrate: if only legacy providers config exists, convert to model_list
	if len(cfg.ModelList) == 0 && cfg.HasProvidersConfig() {
		cfg.ModelList = ConvertProvidersToModelList(cfg)
	}

	// Validate model_list for uniqueness and required fields
	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Agents.Defaults.Workspace)
}

func (c *Config) GetAPIKey() string {
	if c.Providers.OpenRouter.APIKey != "" {
		return c.Providers.OpenRouter.APIKey
	}
	if c.Providers.Anthropic.APIKey != "" {
		return c.Providers.Anthropic.APIKey
	}
	if c.Providers.OpenAI.APIKey != "" {
		return c.Providers.OpenAI.APIKey
	}
	if c.Providers.Gemini.APIKey != "" {
		return c.Providers.Gemini.APIKey
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIKey
	}
	if c.Providers.Groq.APIKey != "" {
		return c.Providers.Groq.APIKey
	}
	if c.Providers.VLLM.APIKey != "" {
		return c.Providers.VLLM.APIKey
	}
	if c.Providers.ShengSuanYun.APIKey != "" {
		return c.Providers.ShengSuanYun.APIKey
	}
	if c.Providers.Cerebras.APIKey != "" {
		return c.Providers.Cerebras.APIKey
	}
	return ""
}

func (c *Config) GetAPIBase() string {
	if c.Providers.OpenRouter.APIKey != "" {
		if c.Providers.OpenRouter.APIBase != "" {
			return c.Providers.OpenRouter.APIBase
		}
		return "https://openrouter.ai/api/v1"
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIBase
	}
	if c.Providers.VLLM.APIKey != "" && c.Providers.VLLM.APIBase != "" {
		return c.Providers.VLLM.APIBase
	}
	return ""
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

// GetModelConfig returns the ModelConfig for the given model name.
// If multiple configs exist with the same model_name, it uses round-robin
// selection for load balancing. Returns an error if the model is not found.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	matches := c.findMatches(modelName)
	if len(matches) == 0 {
		return nil, fmt.Errorf("model %q not found in model_list or providers", modelName)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	idx := rrCounter.Add(1) % uint64(len(matches))
	return &matches[idx], nil
}

// findMatches finds all ModelConfig entries with the given model_name.
func (c *Config) findMatches(modelName string) []ModelConfig {
	var matches []ModelConfig
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			matches = append(matches, c.ModelList[i])
		}
	}
	return matches
}

// HasProvidersConfig checks if any provider in the old providers config has configuration.
//
//nolint:gocyclo // exhaustive provider field check; one condition per provider
func (c *Config) HasProvidersConfig() bool {
	v := c.Providers
	return v.Anthropic.APIKey != "" || v.Anthropic.APIBase != "" ||
		v.OpenAI.APIKey != "" || v.OpenAI.APIBase != "" ||
		v.OpenRouter.APIKey != "" || v.OpenRouter.APIBase != "" ||
		v.Groq.APIKey != "" || v.Groq.APIBase != "" ||
		v.Zhipu.APIKey != "" || v.Zhipu.APIBase != "" ||
		v.VLLM.APIKey != "" || v.VLLM.APIBase != "" ||
		v.Gemini.APIKey != "" || v.Gemini.APIBase != "" ||
		v.Nvidia.APIKey != "" || v.Nvidia.APIBase != "" ||
		v.Ollama.APIKey != "" || v.Ollama.APIBase != "" ||
		v.Moonshot.APIKey != "" || v.Moonshot.APIBase != "" ||
		v.ShengSuanYun.APIKey != "" || v.ShengSuanYun.APIBase != "" ||
		v.DeepSeek.APIKey != "" || v.DeepSeek.APIBase != "" ||
		v.Cerebras.APIKey != "" || v.Cerebras.APIBase != "" ||
		v.VolcEngine.APIKey != "" || v.VolcEngine.APIBase != "" ||
		v.GitHubCopilot.APIKey != "" || v.GitHubCopilot.APIBase != "" ||
		v.Antigravity.APIKey != "" || v.Antigravity.APIBase != "" ||
		v.Qwen.APIKey != "" || v.Qwen.APIBase != "" ||
		v.Mistral.APIKey != "" || v.Mistral.APIBase != ""
}

// ValidateModelList validates all ModelConfig entries in the model_list.
// It checks that each model config is valid.
// Note: Multiple entries with the same model_name are allowed for load balancing.
func (c *Config) ValidateModelList() error {
	for i := range c.ModelList {
		if err := c.ModelList[i].Validate(); err != nil {
			return fmt.Errorf("model_list[%d]: %w", i, err)
		}
	}
	return nil
}
