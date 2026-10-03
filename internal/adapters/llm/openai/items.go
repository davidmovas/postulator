package openai

import (
	port "github.com/davidmovas/postulator/internal/application/llm"
)

const (
	itemFunctionCall   = "function_call"
	itemFunctionOutput = "function_call_output"
)

type inputItem struct {
	Type      string `json:"type,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Output    string `json:"output,omitempty"`
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
			Name:      message.Call.Name,
			Arguments: string(message.Call.Args),
		}
	case message.Result != nil:
		return inputItem{
			Type:   itemFunctionOutput,
			CallID: message.Result.CallID,
			Output: string(message.Result.Output),
		}
	default:
		return inputItem{Role: string(message.Role), Content: message.Text}
	}
}
