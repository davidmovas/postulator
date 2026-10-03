package llm

type Role string

const (
	RoleWriter Role = "writer"
	RoleEditor Role = "editor"
	RoleLinker Role = "linker"
	RoleJudge  Role = "judge"
	RoleChat   Role = "chat"
	RoleImage  Role = "image"
	RoleTitler Role = "titler"
)

func Roles() []Role {
	return []Role{RoleWriter, RoleEditor, RoleLinker, RoleJudge, RoleChat, RoleImage, RoleTitler}
}

func (r Role) Valid() bool {
	switch r {
	case RoleWriter, RoleEditor, RoleLinker, RoleJudge, RoleChat, RoleImage, RoleTitler:
		return true
	default:
		return false
	}
}

const ProviderOpenAI = "openai"

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func (r ModelRef) Valid() bool {
	return r.Provider != "" && r.Model != ""
}

func (r ModelRef) Supported() bool {
	return r.Provider == ProviderOpenAI && r.Model != ""
}

func (r ModelRef) String() string {
	return r.Provider + ":" + r.Model
}

func SecretRef(provider string) string {
	return "llm:" + provider + ":api_key"
}

type ReasoningEffort string

const (
	EffortNone   ReasoningEffort = "none"
	EffortLow    ReasoningEffort = "low"
	EffortMedium ReasoningEffort = "medium"
	EffortHigh   ReasoningEffort = "high"
	EffortXHigh  ReasoningEffort = "xhigh"
)

func (e ReasoningEffort) Valid() bool {
	switch e {
	case EffortNone, EffortLow, EffortMedium, EffortHigh, EffortXHigh:
		return true
	default:
		return false
	}
}

const (
	allowanceLow    = 2048
	allowanceMedium = 8192
	allowanceHigh   = 24576
	allowanceXHigh  = 49152
)

func Allowance(effort ReasoningEffort) int {
	switch effort {
	case EffortNone:
		return 0
	case EffortLow:
		return allowanceLow
	case EffortHigh:
		return allowanceHigh
	case EffortXHigh:
		return allowanceXHigh
	case EffortMedium:
		return allowanceMedium
	default:
		return allowanceMedium
	}
}

type ServiceTier string

const (
	TierDefault ServiceTier = "default"
	TierFlex    ServiceTier = "flex"
)

func (t ServiceTier) Valid() bool {
	switch t {
	case TierDefault, TierFlex:
		return true
	default:
		return false
	}
}

type ModelInfo struct {
	Ref                    ModelRef        `json:"ref"`
	ReasoningEffort        ReasoningEffort `json:"reasoningEffort,omitempty"`
	ContextTokens          int             `json:"contextTokens"`
	MaxOutputTokens        int             `json:"maxOutputTokens"`
	InputUSDPerM           float64         `json:"inputUsdPerM"`
	CachedInputUSDPerM     float64         `json:"cachedInputUsdPerM,omitempty"`
	CacheWriteUSDPerM      float64         `json:"cacheWriteUsdPerM,omitempty"`
	OutputUSDPerM          float64         `json:"outputUsdPerM"`
	FlexInputUSDPerM       float64         `json:"flexInputUsdPerM,omitempty"`
	FlexCachedInputUSDPerM float64         `json:"flexCachedInputUsdPerM,omitempty"`
	FlexCacheWriteUSDPerM  float64         `json:"flexCacheWriteUsdPerM,omitempty"`
	FlexOutputUSDPerM      float64         `json:"flexOutputUsdPerM,omitempty"`
	RPM                    int             `json:"rpm"`
	TPM                    int             `json:"tpm"`
	SupportsStructured     bool            `json:"supportsStructured"`
	SupportsImages         bool            `json:"supportsImages"`
	Reasoning              bool            `json:"reasoning"`
}

func (i ModelInfo) OffersFlex() bool {
	return i.FlexInputUSDPerM > 0
}

type Usage struct {
	Input       int `json:"input"`
	CachedInput int `json:"cachedInput"`
	CacheWrite  int `json:"cacheWrite"`
	Output      int `json:"output"`
	Reasoning   int `json:"reasoning"`
	Total       int `json:"total"`
}

func (u Usage) Add(other Usage) Usage {
	return Usage{
		Input:       u.Input + other.Input,
		CachedInput: u.CachedInput + other.CachedInput,
		CacheWrite:  u.CacheWrite + other.CacheWrite,
		Output:      u.Output + other.Output,
		Reasoning:   u.Reasoning + other.Reasoning,
		Total:       u.Total + other.Total,
	}
}

const tokensPerMillion = 1_000_000

type prices struct {
	input  float64
	cached float64
	write  float64
	output float64
}

func pricesOf(info ModelInfo, tier ServiceTier) prices {
	chosen := prices{
		input: info.InputUSDPerM, cached: info.CachedInputUSDPerM, write: info.CacheWriteUSDPerM,
		output: info.OutputUSDPerM,
	}
	if tier == TierFlex && info.OffersFlex() {
		chosen = prices{
			input: info.FlexInputUSDPerM, cached: info.FlexCachedInputUSDPerM, write: info.FlexCacheWriteUSDPerM,
			output: info.FlexOutputUSDPerM,
		}
	}
	if chosen.cached <= 0 {
		chosen.cached = chosen.input
	}
	if chosen.write <= 0 {
		chosen.write = chosen.input
	}
	return chosen
}

func Cost(usage Usage, info ModelInfo, tier ServiceTier) float64 {
	rate := pricesOf(info, tier)
	read := min(max(usage.CachedInput, 0), usage.Input)
	written := min(max(usage.CacheWrite, 0), usage.Input-read)
	fresh := usage.Input - read - written

	return perMillion(fresh, rate.input) + perMillion(read, rate.cached) + perMillion(written, rate.write) +
		perMillion(usage.Output, rate.output)
}

func perMillion(tokens int, usdPerMillion float64) float64 {
	return float64(tokens) / tokensPerMillion * usdPerMillion
}
