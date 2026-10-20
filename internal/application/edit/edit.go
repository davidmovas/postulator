package edit

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonNoWriter  = "field_not_writable"
	ReasonStoreOwns = "field_owned_by_store"
	ReasonNoPlugin  = "field_needs_plugin"
)

type Outcome string

const (
	OutcomeWritten Outcome = "written"
	OutcomeMoved   Outcome = "moved"
	OutcomeHeld    Outcome = "held"
)

type Target struct {
	Type    string
	ID      int64
	Product bool
}

type Write struct {
	Value    *string
	Expected *string
	Social   *Social
	Key      field.Key
}

type Social struct {
	Value    string
	Expected string
}

type MetaWrite struct {
	Value    *string
	Expected *string
}

type Stored struct {
	Value *string
	Hash  *string
}

type Result struct {
	Key     field.Key `json:"key"`
	Outcome Outcome   `json:"outcome"`
	Value   *string   `json:"value,omitempty"`
	Hash    *string   `json:"hash,omitempty"`
}

type CoreWriter interface {
	ReadCore(ctx context.Context, target Target) (map[field.Key]string, error)
	WriteCore(ctx context.Context, target Target, values map[field.Key]string) (map[field.Key]string, error)
}

type StoreWriter interface {
	ReadStore(ctx context.Context, target Target) (map[field.Key]string, error)
	WriteStore(ctx context.Context, target Target, values map[field.Key]string) (map[field.Key]string, error)
}

type PluginWriter interface {
	Plugin(ctx context.Context) (present bool, err error)
	WriteBody(ctx context.Context, target Target, content, expected string) (stored Stored, moved *string, err error)
	WriteSEO(ctx context.Context, target Target, values, expected map[string]string) (hashes, moved map[string]string, err error)
	WriteMeta(ctx context.Context, target Target, writes map[string]MetaWrite) (stored map[string]Stored, moved map[string]*string, err error)
}

type Writers struct {
	Core   CoreWriter
	Store  StoreWriter
	Plugin PluginWriter
}

type Service struct {
	writers Writers
}

func New(writers Writers) Service {
	return Service{writers: writers}
}

type route int

const (
	routeCore route = iota
	routeStore
	routeMeta
	routeSEO
	routeBody
	routes
)

var seoNames = map[field.Key]string{field.SEOTitle: "title", field.SEODescription: "description", field.SEOCanonical: "canonical"}

var socialNames = map[field.Key]string{field.SEOTitle: "ogTitle", field.SEODescription: "ogDescription"}

func (s Service) Write(ctx context.Context, target Target, writes []Write) ([]Result, error) {
	grouped := make([][]Write, routes)
	for i := range writes {
		at, err := routeOf(target, writes[i])
		if err != nil {
			return nil, err
		}
		grouped[at] = append(grouped[at], writes[i])
	}
	if err := s.settlePlugin(ctx, target, grouped); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(writes))
	stopped := false
	for at := range routes {
		group := grouped[at]
		if len(group) == 0 {
			continue
		}
		if stopped {
			results = append(results, held(group)...)
			continue
		}
		written, err := s.write(ctx, target, at, group)
		if err != nil {
			return results, err
		}
		results = append(results, written...)
		stopped = anyMoved(written)
	}
	return results, nil
}

func routeOf(target Target, write Write) (route, error) {
	key := write.Key
	if key.Kind() == field.Meta {
		if !field.MetaKeyAllowed(key.Instance()) {
			return 0, refused(key, field.ReasonDenied, "the meta key is protected, the store's own, or not a plain key")
		}
		return routeMeta, nil
	}
	spec, known := field.Lookup(key)
	switch {
	case !known || spec.Denied:
		return 0, refused(key, field.ReasonDenied, "the field is never written")
	case target.Product && (key == field.Title || key == field.Excerpt || key == field.FeaturedImage):
		return 0, refused(key, ReasonStoreOwns, "the store owns this field of a product")
	case write.Social != nil && socialNames[key] == "":
		return 0, refused(key, ReasonNoWriter, "only the SEO title and description carry an Open Graph twin")
	}
	switch {
	case key == field.Title, key == field.Excerpt, key == field.FeaturedImage:
		return routeCore, nil
	case key == field.ProductShortDescription && target.Product:
		return routeStore, nil
	case key == field.Body && !target.Product, key == field.ProductDescription && target.Product:
		return routeBody, nil
	case seoNames[key] != "":
		return routeSEO, nil
	default:
		return 0, refused(key, ReasonNoWriter, "no writer takes this field here")
	}
}

func (s Service) settlePlugin(ctx context.Context, target Target, grouped [][]Write) error {
	needed := append(append(append([]Write{}, grouped[routeMeta]...), grouped[routeSEO]...), grouped[routeBody]...)
	if len(needed) == 0 {
		return nil
	}
	present, err := s.writers.Plugin.Plugin(ctx)
	switch {
	case err != nil:
		return err
	case present:
		return nil
	case len(grouped[routeMeta])+len(grouped[routeSEO]) > 0 || target.Product:
		return refused(needed[0].Key, ReasonNoPlugin, "the field is written through the companion plugin, which the site does not have")
	}
	grouped[routeCore] = append(grouped[routeCore], grouped[routeBody]...)
	grouped[routeBody] = nil
	return nil
}

func (s Service) write(ctx context.Context, target Target, at route, group []Write) ([]Result, error) {
	switch at {
	case routeCore:
		return checked(ctx, target, group, s.writers.Core.ReadCore, s.writers.Core.WriteCore)
	case routeStore:
		return checked(ctx, target, group, s.writers.Store.ReadStore, s.writers.Store.WriteStore)
	case routeMeta:
		return s.writeMeta(ctx, target, group)
	case routeSEO:
		return s.writeSEO(ctx, target, group)
	default:
		return s.writeBody(ctx, target, group)
	}
}

type reader func(context.Context, Target) (map[field.Key]string, error)

type writer func(context.Context, Target, map[field.Key]string) (map[field.Key]string, error)

func checked(ctx context.Context, target Target, group []Write, read reader, put writer) ([]Result, error) {
	live, err := read(ctx, target)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(group))
	values := make(map[field.Key]string, len(group))
	for i := range group {
		current := live[group[i].Key]
		switch {
		case holds(current, group[i].Expected):
			values[group[i].Key] = text(group[i].Value)
		case current == text(group[i].Value):
		default:
			results = append(results, Result{Key: group[i].Key, Outcome: OutcomeMoved, Hash: new(block.BodyHash(current))})
		}
	}
	if len(results) > 0 {
		return withHeld(results, group), nil
	}

	stored := live
	if len(values) > 0 {
		if stored, err = put(ctx, target, values); err != nil {
			return nil, err
		}
	}
	for i := range group {
		value := stored[group[i].Key]
		results = append(results, Result{Key: group[i].Key, Outcome: OutcomeWritten, Value: &value, Hash: new(block.BodyHash(value))})
	}
	return results, nil
}

const sendings = 2

func (s Service) writeMeta(ctx context.Context, target Target, group []Write) ([]Result, error) {
	written := make(map[field.Key]Result, len(group))
	pending := group
	for range sendings {
		if len(pending) == 0 {
			break
		}
		writes := make(map[string]MetaWrite, len(pending))
		for i := range pending {
			writes[pending[i].Key.Instance()] = MetaWrite{Value: pending[i].Value, Expected: pending[i].Expected}
		}
		stored, moved, err := s.writers.Plugin.WriteMeta(ctx, target, writes)
		if err != nil {
			return nil, err
		}
		next := make([]Write, 0, len(pending))
		shifted := make([]Result, 0)
		for i := range pending {
			hash, isMoved := moved[pending[i].Key.Instance()]
			switch {
			case len(moved) == 0:
				held := stored[pending[i].Key.Instance()]
				written[pending[i].Key] = Result{Key: pending[i].Key, Outcome: OutcomeWritten, Value: held.Value, Hash: held.Hash}
			case !isMoved:
				next = append(next, pending[i])
			case sameStored(hash, pending[i].Value):
				written[pending[i].Key] = Result{Key: pending[i].Key, Outcome: OutcomeWritten, Value: pending[i].Value, Hash: hash}
			default:
				shifted = append(shifted, Result{Key: pending[i].Key, Outcome: OutcomeMoved, Hash: hash})
			}
		}
		if len(shifted) > 0 {
			return settled(group, written, shifted), nil
		}
		pending = next
	}
	return settled(group, written, nil), nil
}

func (s Service) writeSEO(ctx context.Context, target Target, group []Write) ([]Result, error) {
	written := make(map[field.Key]Result, len(group))
	pending := group
	for range sendings {
		if len(pending) == 0 {
			break
		}
		values, expected := seoRequest(pending)
		hashes, moved, err := s.writers.Plugin.WriteSEO(ctx, target, values, expected)
		if err != nil {
			return nil, err
		}
		next := make([]Write, 0, len(pending))
		shifted := make([]Result, 0)
		for i := range pending {
			name, value := seoNames[pending[i].Key], values[seoNames[pending[i].Key]]
			hash, isMoved := movedSEO(moved, pending[i])
			switch {
			case len(moved) == 0:
				written[pending[i].Key] = Result{Key: pending[i].Key, Outcome: OutcomeWritten, Value: &value, Hash: new(hashes[name])}
			case !isMoved:
				next = append(next, pending[i])
			case seoStored(moved, values, pending[i]):
				written[pending[i].Key] = Result{Key: pending[i].Key, Outcome: OutcomeWritten, Value: &value, Hash: new(block.BodyHash(value))}
			default:
				shifted = append(shifted, Result{Key: pending[i].Key, Outcome: OutcomeMoved, Hash: &hash})
			}
		}
		if len(shifted) > 0 {
			return settled(group, written, shifted), nil
		}
		pending = next
	}
	return settled(group, written, nil), nil
}

func seoRequest(group []Write) (values, expected map[string]string) {
	values = make(map[string]string, 2*len(group))
	expected = make(map[string]string, 2*len(group))
	for i := range group {
		name := seoNames[group[i].Key]
		values[name], expected[name] = text(group[i].Value), expectation(group[i].Expected)
		if social := group[i].Social; social != nil {
			twin := socialNames[group[i].Key]
			values[twin], expected[twin] = social.Value, social.Expected
		}
	}
	return values, expected
}

func movedSEO(moved map[string]string, write Write) (string, bool) {
	if hash, shifted := moved[seoNames[write.Key]]; shifted {
		return hash, true
	}
	if write.Social == nil {
		return "", false
	}
	hash, shifted := moved[socialNames[write.Key]]
	return hash, shifted
}

func seoStored(moved, values map[string]string, write Write) bool {
	for _, name := range []string{seoNames[write.Key], socialNames[write.Key]} {
		if hash, shifted := moved[name]; shifted && (name == "" || hash != block.BodyHash(values[name])) {
			return false
		}
	}
	return true
}

func (s Service) writeBody(ctx context.Context, target Target, group []Write) ([]Result, error) {
	results := make([]Result, 0, len(group))
	for i := range group {
		content := text(group[i].Value)
		stored, moved, err := s.writers.Plugin.WriteBody(ctx, target, content, expectation(group[i].Expected))
		switch {
		case err != nil:
			return nil, err
		case moved != nil && *moved == block.BodyHash(content):
			results = append(results, Result{Key: group[i].Key, Outcome: OutcomeWritten, Value: &content, Hash: moved})
		case moved != nil:
			results = append(results, Result{Key: group[i].Key, Outcome: OutcomeMoved, Hash: moved})
			return withHeld(results, group), nil
		default:
			results = append(results, Result{Key: group[i].Key, Outcome: OutcomeWritten, Value: stored.Value, Hash: stored.Hash})
		}
	}
	return results, nil
}

func sameStored(hash, value *string) bool {
	if hash == nil || value == nil {
		return hash == nil && value == nil
	}
	return *hash == block.BodyHash(*value)
}

func settled(group []Write, written map[field.Key]Result, shifted []Result) []Result {
	results := make([]Result, 0, len(group))
	for i := range group {
		if held, done := written[group[i].Key]; done {
			results = append(results, held)
			continue
		}
		outcome := Result{Key: group[i].Key, Outcome: OutcomeHeld}
		for k := range shifted {
			if shifted[k].Key == group[i].Key {
				outcome = shifted[k]
			}
		}
		results = append(results, outcome)
	}
	return results
}

func holds(current string, expected *string) bool {
	if expected == nil {
		return current == ""
	}
	return block.BodyHash(current) == *expected
}

func expectation(expected *string) string {
	if expected == nil {
		return block.BodyHash("")
	}
	return *expected
}

func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func held(group []Write) []Result {
	return withHeld(make([]Result, 0, len(group)), group)
}

func withHeld(results []Result, group []Write) []Result {
	for i := range group {
		listed := false
		for k := range results {
			listed = listed || results[k].Key == group[i].Key
		}
		if !listed {
			results = append(results, Result{Key: group[i].Key, Outcome: OutcomeHeld})
		}
	}
	return results
}

func anyMoved(results []Result) bool {
	for i := range results {
		if results[i].Outcome == OutcomeMoved {
			return true
		}
	}
	return false
}

func refused(key field.Key, reason, message string) error {
	return errors.New(errors.Invalid, message).WithDetail("reason", reason).WithDetail("key", string(key))
}
