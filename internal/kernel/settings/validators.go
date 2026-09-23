package settings

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Constraints struct {
	Min      any
	Max      any
	Enum     []string
	NonEmpty bool
}

type Validator[T any] struct {
	check    func(T) error
	describe func(*Constraints)
}

func Check[T any](fn func(T) error) Validator[T] {
	return Validator[T]{check: fn}
}

func (v Validator[T]) describing(fn func(*Constraints)) Validator[T] {
	v.describe = fn
	return v
}

func IntMin(minimum int) Validator[int] {
	return Check(func(value int) error {
		if value < minimum {
			return fmt.Errorf("value %d is below the minimum %d", value, minimum)
		}
		return nil
	}).describing(func(c *Constraints) { c.Min = minimum })
}

func IntMax(maximum int) Validator[int] {
	return Check(func(value int) error {
		if value > maximum {
			return fmt.Errorf("value %d is above the maximum %d", value, maximum)
		}
		return nil
	}).describing(func(c *Constraints) { c.Max = maximum })
}

func IntRange(minimum, maximum int) Validator[int] {
	below, above := IntMin(minimum), IntMax(maximum)
	return Check(func(value int) error {
		if err := below.check(value); err != nil {
			return err
		}
		return above.check(value)
	}).describing(func(c *Constraints) { c.Min, c.Max = minimum, maximum })
}

func DurationRange(minimum, maximum time.Duration) Validator[time.Duration] {
	return Check(func(value time.Duration) error {
		if value < minimum {
			return fmt.Errorf("value %s is below the minimum %s", value, minimum)
		}
		if value > maximum {
			return fmt.Errorf("value %s is above the maximum %s", value, maximum)
		}
		return nil
	}).describing(func(c *Constraints) { c.Min, c.Max = minimum.String(), maximum.String() })
}

func NonEmpty() Validator[string] {
	return Check(func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("value must not be empty")
		}
		return nil
	}).describing(func(c *Constraints) { c.NonEmpty = true })
}

func OneOf(allowed []string) Validator[string] {
	return Check(func(value string) error {
		if !slices.Contains(allowed, value) {
			return fmt.Errorf("value %q is not one of %s", value, strings.Join(allowed, ", "))
		}
		return nil
	}).describing(func(c *Constraints) { c.Enum = slices.Clone(allowed) })
}
