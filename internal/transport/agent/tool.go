package agent

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/log"
)

const resultKey = "result"

func explain(err error) error {
	var known *errors.Error
	if !stderrors.As(err, &known) || len(known.Details) == 0 {
		return err
	}

	keys := make([]string, 0, len(known.Details))
	for key := range known.Details {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	named := make([]string, 0, len(keys))
	for _, key := range keys {
		value := known.Details[key]
		if log.IsSensitiveKey(key) {
			value = log.Mask
		}
		named = append(named, key+": "+fmt.Sprint(value))
	}
	return errors.New(known.Code, err.Error()+" ("+strings.Join(named, ", ")+")")
}

func objectOf(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "the tool answered something that cannot be encoded")
	}

	decoded, err := decoded(encoded)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "the tool answer could not be read back")
	}
	if object, ok := decoded.(map[string]any); ok {
		return object, nil
	}
	return map[string]any{resultKey: decoded}, nil
}

func decoded(encoded []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func encode(value map[string]any) json.RawMessage {
	if value == nil {
		return json.RawMessage(emptyArguments)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(emptyArguments)
	}
	return encoded
}
