package openai

import (
	"encoding/json"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

const (
	itemFunctionCall     = "function_call"
	itemFunctionOutput   = "function_call_output"
	itemToolSearchCall   = "tool_search_call"
	itemToolSearchOutput = "tool_search_output"
)

type inputItem struct {
	Type      string          `json:"type,omitempty"`
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Namespace string          `json:"namespace,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments any             `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
	Tools     json.RawMessage `json:"tools,omitempty"`
	Execution string          `json:"execution,omitempty"`
}

func itemsOf(messages []port.Message) []inputItem {
	items := make([]inputItem, 0, len(messages))
	for _, message := range messages {
		items = append(items, itemOf(message))
	}
	return items
}

func itemOf(message port.Message) inputItem {
	switch {
	case message.Call != nil:
		return inputItem{
			Type:      itemFunctionCall,
			CallID:    message.Call.ID,
			Namespace: message.Call.Namespace,
			Name:      message.Call.Name,
			Arguments: string(message.Call.Args),
		}
	case message.Result != nil:
		return inputItem{
			Type:   itemFunctionOutput,
			CallID: message.Result.CallID,
			Output: string(message.Result.Output),
		}
	case message.Search != nil:
		return searchItemOf(*message.Search)
	default:
		return inputItem{Role: string(message.Role), Content: message.Text}
	}
}

func searchItemOf(search port.ToolSearch) inputItem {
	item := inputItem{CallID: search.CallID, Execution: search.Execution}
	if search.Kind == port.SearchOutput {
		item.Type, item.Tools = itemToolSearchOutput, search.Payload
		return item
	}
	item.Type, item.Arguments = itemToolSearchCall, search.Payload
	return item
}
