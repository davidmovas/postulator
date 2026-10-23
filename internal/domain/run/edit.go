package run

import (
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ParamEdit = "edit"

type EditOp string

const (
	EditSetField    EditOp = "set_field"
	EditReplace     EditOp = "replace"
	EditInsertAfter EditOp = "insert_after"
	EditDelete      EditOp = "delete"
	EditPlaceLink   EditOp = "place_link"
)

type Edit struct {
	Op       EditOp `json:"op"`
	Field    string `json:"field,omitempty"`
	Value    string `json:"value,omitempty"`
	Hash     string `json:"hash,omitempty"`
	HTML     string `json:"html,omitempty"`
	TargetID string `json:"targetId,omitempty"`
	Anchor   string `json:"anchor,omitempty"`
	Index    int    `json:"index"`
}

func (e Edit) Block() bool {
	return e.Op != EditSetField
}

func (e Edit) Validate() error {
	if e.Block() && strings.TrimSpace(e.Hash) == "" {
		return invalid("a block edit names the hash of the body it was read under, so a change since is refused", "hash")
	}
	switch e.Op {
	case EditSetField:
		return e.validField()
	case EditReplace, EditDelete:
		if e.Index < 0 {
			return invalid("name the block by the index pages_blocks listed", "index")
		}
	case EditInsertAfter:
		if e.Index < -1 {
			return invalid("name the block to insert after, or -1 for the start", "index")
		}
	case EditPlaceLink:
		if e.Index < 0 || strings.TrimSpace(e.TargetID) == "" || strings.TrimSpace(e.Anchor) == "" {
			return invalid("a link names its block, the page it points to and the text that becomes it", "anchor")
		}
	default:
		return invalid("the edit's op is not recognized", "op")
	}
	if (e.Op == EditReplace || e.Op == EditInsertAfter) && strings.TrimSpace(e.HTML) == "" {
		return invalid("a block is replaced or inserted with the html it takes", "html")
	}
	return nil
}

func (e Edit) validField() error {
	key := field.Key(e.Field)
	spec, known := field.Lookup(key)
	switch {
	case !known:
		return invalid("the field is not recognized", "field").WithDetail("key", e.Field)
	case spec.Denied || spec.Write == field.None:
		return invalid("the field is the store's or WordPress's own and is never written", "field").WithDetail("key", e.Field)
	case spec.Person:
		return invalid("the field is a person's switch and is set on the page by a person", "field").WithDetail("key", e.Field)
	case key == field.Body || key == field.ProductDescription:
		return invalid("the body is changed block by block, never rewritten whole", "field").WithDetail("key", e.Field)
	case strings.TrimSpace(e.Value) == "":
		return invalid("a field is set to the value it takes", "value")
	}
	return nil
}

func EditRecipe(e Edit) []template.StepSpec {
	return []template.StepSpec{{Name: string(StepEditPage), Enabled: true, Params: map[string]any{ParamEdit: e}}}
}

func EditOf(params map[string]any) (Edit, error) {
	held, ok := params[ParamEdit]
	if !ok {
		return Edit{}, invalid("the edit run carries no edit", "params")
	}
	encoded, err := json.Marshal(held)
	if err != nil {
		return Edit{}, errors.Wrap(err, errors.Internal, "encode the edit the run carries")
	}
	var out Edit
	if err = json.Unmarshal(encoded, &out); err != nil {
		return Edit{}, invalid("the edit the run carries cannot be read", "params")
	}
	return out, nil
}
