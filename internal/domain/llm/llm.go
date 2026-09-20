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

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func (r ModelRef) Valid() bool {
	return r.Provider != "" && r.Model != ""
}

func (r ModelRef) String() string {
	return r.Provider + ":" + r.Model
}

func SecretRef(provider string) string {
	return "llm:" + provider + ":api_key"
}

type ModelInfo struct {
	Ref                ModelRef `json:"ref"`
	ContextTokens      int      `json:"contextTokens"`
	MaxOutputTokens    int      `json:"maxOutputTokens"`
	InputUSDPerM       float64  `json:"inputUsdPerM"`
	OutputUSDPerM      float64  `json:"outputUsdPerM"`
	RPM                int      `json:"rpm"`
	TPM                int      `json:"tpm"`
	SupportsStructured bool     `json:"supportsStructured"`
	SupportsImages     bool     `json:"supportsImages"`
	Reasoning          bool     `json:"reasoning"`
}

type Usage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

func (u Usage) Add(other Usage) Usage {
	return Usage{Input: u.Input + other.Input, Output: u.Output + other.Output, Total: u.Total + other.Total}
}

const tokensPerMillion = 1_000_000

func Cost(usage Usage, info ModelInfo) float64 {
	input := float64(usage.Input) / tokensPerMillion * info.InputUSDPerM
	output := float64(usage.Output) / tokensPerMillion * info.OutputUSDPerM
	return input + output
}
