package run

import "github.com/davidmovas/postulator/internal/domain/content"

type EstimateFinding struct {
	Severity content.Severity `json:"severity"`
	Code     string           `json:"code"`
	Message  string           `json:"message"`
	PageID   string           `json:"pageId,omitempty"`
	Path     string           `json:"path,omitempty"`
}

type Estimate struct {
	Tokens   int               `json:"tokens"`
	USD      float64           `json:"usd"`
	Findings []EstimateFinding `json:"findings"`
}

func (e Estimate) Blocking() []EstimateFinding {
	out := make([]EstimateFinding, 0)
	for i := range e.Findings {
		if e.Findings[i].Severity == content.SeverityError {
			out = append(out, e.Findings[i])
		}
	}
	return out
}
