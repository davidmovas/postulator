package steps

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameEditPage = string(run.StepEditPage)

	checkpointEditPage = "edit_page.state"

	ReasonEditBodyMoved   = "the body changed on the site since the agent listed its blocks; sync the site and list them again"
	ReasonEditFieldMoved  = "the field changed on the site, or in the model, since the agent read it; sync the site and read it again"
	ReasonEditClientBlock = "the block is the owner's own, and the page type's body does not rewrite the owner's text; " +
		"insert a block beside it, or switch rewrites on for the type first"
	ReasonEditNoCustomField = "the page's type names no such custom field"
	ReasonEditLinkTarget    = "the page the link points to is not on the site"
)

type editState struct {
	wake     time.Time
	Publish  PublishResult  `json:"publish"`
	Write    relinkWrite    `json:"write"`
	Before   string         `json:"before"`
	Provider field.Provider `json:"provider,omitempty"`
	Seq      int            `json:"seq"`
	Planned  bool           `json:"planned"`
	Finished bool           `json:"finished"`
}

func editStateOf(sc *run.StepContext, wake time.Time) (editState, error) {
	held, found, err := run.Get[editState](sc.Check, checkpointEditPage)
	if err != nil {
		return editState{}, err
	}
	if !found || held.Finished || !held.Planned {
		return editState{wake: wake, Seq: held.Seq}, nil
	}
	held.wake = wake
	return held, nil
}

func (s *editState) checkpoint() (run.Checkpoint, error) {
	s.Seq++
	check := run.NewCheckpoint()
	if err := run.Set(check, checkpointEditPage, *s); err != nil {
		return nil, err
	}
	return check, nil
}

func (s *editState) wait(message string) (run.Result, error) {
	s.Planned = true
	check, err := s.checkpoint()
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{Next: run.TransitionWait, WakeAt: s.wake, Checkpoint: check, Message: message}, nil
}

func (s *editState) settle(message string) (run.Result, error) {
	s.Finished = true
	check, err := s.checkpoint()
	if err != nil {
		return run.Result{}, err
	}
	blob, err := encode(s.Publish, "publish result")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{
		Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: blob}}, Checkpoint: check, Message: message,
	}, nil
}

func EditPage(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameEditPage,
		Preflight: pluginPreflight(deps, NameEditPage, "cannot read or write the page's body and stands down"),
		Produces:  []run.ArtifactKind{run.ArtifactPublishResult},
		Retry:     run.RetryPolicy{Max: 2},
		Timeout:   relinkTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			wanted, err := run.EditOf(sc.Params)
			if err != nil {
				return run.Result{}, err
			}
			if refused := wanted.Validate(); refused != nil {
				return run.Result{}, refused
			}
			page, err := deps.Pages.Get(ctx, sc.Item.TargetID)
			if err != nil {
				return run.Result{}, err
			}
			if reason, held := offLimits(page); held {
				return needsHuman("the page " + page.Path + " " + reason), nil
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			state, err := editStateOf(sc, deps.now())
			if err != nil {
				return run.Result{}, err
			}
			switch {
			case state.Planned && wanted.Block():
				return landBlockEdit(ctx, deps, sc, client, page, &state)
			case state.Planned:
				return landFieldEdit(ctx, deps, sc, client, page, wanted, &state)
			case wanted.Block():
				return planBlockEdit(ctx, deps, sc, client, page, wanted, &state)
			default:
				return planFieldEdit(ctx, deps, sc, client, page, wanted, &state)
			}
		},
	}
}

func editedResult(page pagemap.Page) PublishResult {
	return PublishResult{
		WPID: *page.WPID, URL: page.Path, Status: string(page.Status), SEOApplied: make([]string, 0),
		Skipped: make([]string, 0), Findings: make([]content.Finding, 0), Mismatches: make([]Mismatch, 0),
		Fields: make([]field.Change, 0), Blocks: make([]block.Change, 0), Version: publishRecordVersion,
	}
}

func planBlockEdit(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, page pagemap.Page,
	wanted run.Edit, state *editState) (run.Result, error) {
	body, reason, err := readSiteBody(ctx, client, page, ReasonPageGone, ReasonPageUnreadable)
	if err != nil {
		return run.Result{}, err
	}
	switch {
	case reason != "":
		return needsHuman(page.Path + ": " + reason), nil
	case !body.raw.SaveStable:
		return needsHuman(page.Path + ": " + ReasonSaveAltersClient), nil
	case body.raw.ContentHash != wanted.Hash:
		return needsHuman(page.Path + ": " + ReasonEditBodyMoved), nil
	}
	written, err := deps.ourBlocks(ctx, page.ID)
	if err != nil {
		return run.Result{}, err
	}
	applied, refusal, err := deps.applyEdit(ctx, sc, page, block.Parse(body.raw.Content), written, wanted)
	if err != nil || refusal != "" {
		if err != nil {
			return run.Result{}, err
		}
		return needsHuman(page.Path + ": " + refusal), nil
	}
	after := applied.body.Serialize()
	ours := applied.oursIn()
	state.Publish = editedResult(page)
	state.Publish.PreviousContent, state.Publish.PreviousContentHash = body.raw.Content, body.raw.ContentHash
	state.Publish.Blocks = ownedDiff(body.raw.Content, after, written, ours)
	state.Write = relinkWrite{
		Outcome: NeighborResult{
			PageID: page.ID, Path: page.Path, WPID: *page.WPID,
			Before: NeighborBefore{Hash: body.raw.ContentHash, HTML: body.raw.Content},
		},
		After: after, Ours: ours, Placed: applied.placed,
	}
	return state.wait("planned the agent's edit of " + page.Path)
}

func ownedDiff(before, after string, written, ours []string) []block.Change {
	changes := block.Diff(before, after, slices.Concat(written, ours))
	for i := range changes {
		if changes[i].Change == block.ChangeAdded && !slices.Contains(ours, changes[i].Hash) {
			changes[i].Owner = block.OwnerClient
		}
	}
	return changes
}

type appliedEdit struct {
	body   block.Body
	ours   map[string]bool
	placed []block.Placed
}

func (a appliedEdit) oursIn() []string {
	out := make([]string, 0, len(a.ours))
	for i := range a.body.Blocks {
		if hash := a.body.Blocks[i].Hash; a.ours[hash] && !slices.Contains(out, hash) {
			out = append(out, hash)
		}
	}
	return out
}

func (d Deps) applyEdit(ctx context.Context, sc *run.StepContext, page pagemap.Page, current block.Body, written []string,
	wanted run.Edit) (appliedEdit, string, error) {
	applied := appliedEdit{body: current, ours: make(map[string]bool, len(written)+1)}
	for _, hash := range written {
		applied.ours[hash] = true
	}
	at := wanted.Index
	if wanted.Op != run.EditInsertAfter && at >= len(current.Blocks) {
		return appliedEdit{}, "the page holds no block " + strconv.Itoa(at) + "; list its blocks again", nil
	}
	if wanted.Op == run.EditReplace || wanted.Op == run.EditDelete {
		rewrites, err := d.bodyRewrites(ctx, sc, page)
		if err != nil {
			return appliedEdit{}, "", err
		}
		if !applied.ours[current.Blocks[at].Hash] && !rewrites {
			return appliedEdit{}, ReasonEditClientBlock, nil
		}
	}
	html, err := d.authored(ctx, page, current, wanted)
	if err != nil {
		return appliedEdit{}, "", err
	}
	next, err := d.operate(ctx, sc, current, wanted, html, &applied)
	if errors.IsCode(err, errors.Invalid) || errors.IsCode(err, errors.NotFound) {
		return appliedEdit{}, err.Error(), nil
	}
	if err != nil {
		return appliedEdit{}, "", err
	}
	applied.body = next
	return applied, "", nil
}

func (d Deps) bodyRewrites(ctx context.Context, sc *run.StepContext, page pagemap.Page) (bool, error) {
	set, err := d.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return false, err
	}
	scope, _ := set.Scope(set.TypeOf(page).ID, bodyKeyOf(page))
	return scope.Enabled && scope.Rewrites, nil
}

func (d Deps) authored(ctx context.Context, page pagemap.Page, current block.Body, wanted run.Edit) (string, error) {
	if wanted.Op != run.EditReplace && wanted.Op != run.EditInsertAfter {
		return "", nil
	}
	owed, err := d.owesH1(ctx, page.ID)
	if err != nil {
		return "", err
	}
	html, _ := content.SanitizeBlock(wanted.HTML, owed)
	if current.Kind == block.Blocks {
		return block.HTMLBlock(strings.TrimSpace(html)), nil
	}
	return html, nil
}

func (d Deps) operate(ctx context.Context, sc *run.StepContext, current block.Body, wanted run.Edit, html string,
	applied *appliedEdit) (block.Body, error) {
	at := wanted.Index
	switch wanted.Op {
	case run.EditReplace:
		next, err := current.Replace(at, html)
		if err != nil {
			return block.Body{}, err
		}
		delete(applied.ours, current.Blocks[at].Hash)
		applied.ours[next.Blocks[at].Hash] = true
		return next, nil
	case run.EditInsertAfter:
		next, err := current.InsertAfter(at, html)
		if err != nil {
			return block.Body{}, err
		}
		applied.ours[next.Blocks[at+1].Hash] = true
		return next, nil
	case run.EditDelete:
		delete(applied.ours, current.Blocks[at].Hash)
		return current.Delete(at)
	default:
		return d.linked(ctx, sc, current, wanted, applied)
	}
}

func (d Deps) linked(ctx context.Context, sc *run.StepContext, current block.Body, wanted run.Edit,
	applied *appliedEdit) (block.Body, error) {
	target, err := d.Pages.Get(ctx, wanted.TargetID)
	if err != nil {
		return block.Body{}, err
	}
	if target.WPID == nil || target.SiteID != sc.Run.SiteID {
		return block.Body{}, errors.New(errors.Invalid, ReasonEditLinkTarget).WithDetail("targetId", wanted.TargetID)
	}
	owner, err := d.Sites.Get(ctx, sc.Run.SiteID)
	if err != nil {
		return block.Body{}, err
	}
	href := target.Path
	if owner.Settings.LinkURLForm == site.URLFormAbsolute {
		href = pagemap.NewSite(owner.BaseURL).URL(target.Path)
	}
	at := wanted.Index
	next, err := current.PlaceLink(at, block.Link{Anchor: wanted.Anchor, Href: href})
	if err != nil {
		return block.Body{}, err
	}
	before, after := current.Blocks[at].Hash, next.Blocks[at].Hash
	if applied.ours[before] {
		delete(applied.ours, before)
		applied.ours[after] = true
	}
	applied.placed = append(applied.placed, block.Placed{To: target.Path, Anchor: wanted.Anchor, Block: after})
	return next, nil
}

func landBlockEdit(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, page pagemap.Page,
	state *editState) (run.Result, error) {
	write := state.Write
	landed, err := landRelinkedBody(ctx, client, page, write.After, write.Outcome.Before.Hash)
	if err != nil {
		check, checkErr := state.checkpoint()
		if checkErr != nil {
			return run.Result{}, checkErr
		}
		return run.Result{Checkpoint: check}, err
	}
	if !landed.written {
		return needsHuman(page.Path + ": " + ReasonEditBodyMoved), nil
	}
	state.Publish.ContentHash = landed.hash
	state.Publish.Blocks = block.Landed(state.Publish.Blocks, write.After, landed.content)
	record, held := write.landed(landed.content)
	if recordErr := recordSpliced(ctx, deps, page.ID, sc.Run.ID, record); recordErr != nil {
		return run.Result{}, recordErr
	}
	owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	if adoptErr := adopt(ctx, deps, page, pagemap.NewIndex(pages), pagemap.NewSite(owner.BaseURL), held); adoptErr != nil {
		return run.Result{}, adoptErr
	}
	return state.settle("made the agent's edit of " + page.Path)
}

func planFieldEdit(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, page pagemap.Page,
	wanted run.Edit, state *editState) (run.Result, error) {
	key := field.Key(wanted.Field)
	if resaves(key) {
		raw, err := client.GetRaw(ctx, onSiteType(page), *page.WPID)
		switch {
		case wp.IsPluginMissing(err):
		case err != nil:
			return run.Result{}, err
		case !raw.SaveStable:
			return needsHuman(page.Path + ": " + ReasonSaveAltersClient), nil
		}
	}
	if key.Kind() == field.Meta {
		set, err := deps.Types.Set(ctx, sc.Run.SiteID)
		if err != nil {
			return run.Result{}, err
		}
		scope, named := set.Scope(set.TypeOf(page).ID, key)
		if !named || !scope.Enabled || scope.Provider == field.ProviderNone {
			return needsHuman(page.Path + ": " + ReasonEditNoCustomField), nil
		}
		state.Provider = scope.Provider
	}
	before, err := deps.modelValue(ctx, page.ID, key)
	if err != nil {
		return run.Result{}, err
	}
	if field.Hash(before) != wanted.Hash {
		return needsHuman(page.Path + ": " + ReasonEditFieldMoved), nil
	}
	state.Before = before
	state.Publish = editedResult(page)
	return state.wait("planned the agent's edit of the " + wanted.Field + " of " + page.Path)
}

func resaves(key field.Key) bool {
	switch key {
	case field.Title, field.Excerpt, field.ProductShortDescription, field.Upsells, field.CrossSells:
		return true
	default:
		return false
	}
}

func (d Deps) modelValue(ctx context.Context, pageID string, key field.Key) (string, error) {
	if d.Fields == nil {
		return "", nil
	}
	values, err := d.Fields.ListForPage(ctx, pageID)
	if err != nil {
		return "", err
	}
	for i := range values {
		if values[i].Key == key {
			return values[i].Current, nil
		}
	}
	return "", nil
}

func landFieldEdit(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, page pagemap.Page,
	wanted run.Edit, state *editState) (run.Result, error) {
	key := field.Key(wanted.Field)
	value := wanted.Value
	write := edit.Write{Key: key, Value: &value, Provider: state.Provider}
	if wanted.Hash != "" {
		write.Expected = &wanted.Hash
	}
	target := edit.Target{Type: string(onSiteType(page)), ID: *page.WPID, Product: page.WPType == pagemap.WPProduct}
	results, err := editorFor(client).Write(ctx, target, []edit.Write{write})
	if errors.IsCode(err, errors.Invalid) {
		return needsHuman(page.Path + ": " + err.Error()), nil
	}
	if err != nil {
		check, checkErr := state.checkpoint()
		if checkErr != nil {
			return run.Result{}, checkErr
		}
		return run.Result{Checkpoint: check}, err
	}
	stored, written := "", false
	for _, held := range results {
		if held.Key == key && held.Outcome == edit.OutcomeWritten {
			stored, written = value, true
			if held.Value != nil {
				stored = *held.Value
			}
		}
	}
	if !written {
		return needsHuman(page.Path + ": " + ReasonEditFieldMoved), nil
	}
	if recordErr := recordField(ctx, deps, sc, page.ID, key, stored); recordErr != nil {
		return run.Result{}, recordErr
	}
	state.Publish.Fields = []field.Change{field.NewChange(key, state.Before, stored)}
	if state.Provider != field.ProviderNone {
		state.Publish.PreviousFields = []FieldWrite{{Key: key, Before: state.Before, Written: stored, Provider: state.Provider}}
	}
	return state.settle("wrote the agent's " + wanted.Field + " of " + page.Path)
}
