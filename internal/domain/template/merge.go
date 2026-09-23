package template

import (
	"bytes"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func MergePatch(target, patch json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(patch)) == 0 {
		return target, nil
	}

	var patchValue any
	if err := json.Unmarshal(patch, &patchValue); err != nil {
		return nil, invalid("merge patch is not valid json", "patch").WithInternal(err)
	}
	patchObject, ok := patchValue.(map[string]any)
	if !ok {
		return marshal(patchValue)
	}

	var targetValue any
	if len(bytes.TrimSpace(target)) > 0 {
		if err := json.Unmarshal(target, &targetValue); err != nil {
			return nil, invalid("merge target is not valid json", "target").WithInternal(err)
		}
	}
	targetObject, ok := targetValue.(map[string]any)
	if !ok {
		targetObject = map[string]any{}
	}
	return marshal(mergeObjects(targetObject, patchObject))
}

func mergeObjects(target, patch map[string]any) map[string]any {
	for key, value := range patch {
		if value == nil {
			delete(target, key)
			continue
		}
		if child, ok := value.(map[string]any); ok {
			existing, isObject := target[key].(map[string]any)
			if !isObject {
				existing = map[string]any{}
			}
			target[key] = mergeObjects(existing, child)
			continue
		}
		target[key] = value
	}
	return target
}

func marshal(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the merged document")
	}
	return encoded, nil
}

func Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error) {
	document, err := marshal(base)
	if err != nil {
		return TemplateSpec{}, err
	}
	if document, err = MergePatch(document, siteOverride); err != nil {
		return TemplateSpec{}, err
	}
	if document, err = MergePatch(document, pageOverride); err != nil {
		return TemplateSpec{}, err
	}

	var resolved TemplateSpec
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&resolved); err != nil {
		return TemplateSpec{}, invalid("resolved template does not match the template spec", "spec").WithInternal(err)
	}
	if validErr := Validate(resolved); validErr != nil {
		return TemplateSpec{}, validErr
	}
	return resolved, nil
}
