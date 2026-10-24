package field

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type Value struct {
	PageID       string
	Key          Key
	Current      string
	CurrentHash  string
	Base         string
	BaseHash     string
	ObservedAt   time.Time
	WrittenAt    *time.Time
	WrittenRunID *string
}

func Hash(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func NewValue(pageID string, key Key, current string, at time.Time) Value {
	hash := Hash(current)
	return Value{
		PageID: pageID, Key: key, Current: current, CurrentHash: hash, Base: current, BaseHash: hash, ObservedAt: at,
	}
}

func (v Value) Changed() bool {
	return v.CurrentHash != v.BaseHash
}
