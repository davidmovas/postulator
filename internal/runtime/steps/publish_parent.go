package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type placement struct {
	path    string
	wpID    int64
	pending bool
}

func parentFor(ctx context.Context, deps Deps, page pagemap.Page) (placement, error) {
	if !hierarchical(page) || page.WPID != nil {
		return placement{}, nil
	}
	return parentOf(ctx, deps, page)
}

func parentOf(ctx context.Context, deps Deps, page pagemap.Page) (placement, error) {
	if page.ParentPageID == nil {
		above := pagemap.ParentPath(page.Path)
		if above == "" || above == pagemap.RootPath {
			return placement{}, nil
		}
		return placement{}, errors.New(errors.Invalid,
			"the page map holds no page at "+above+", so "+page.Path+" has no parent to sit under").
			WithDetail("pageId", page.ID).
			WithDetail("path", page.Path).
			WithDetail("parentPath", above)
	}

	parent, err := deps.Pages.Get(ctx, *page.ParentPageID)
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return placement{}, errors.New(errors.Invalid,
				"the page names a parent the page map does not hold").
				WithDetail("pageId", page.ID).
				WithDetail("path", page.Path).
				WithDetail("parentPageId", *page.ParentPageID)
		}
		return placement{}, err
	}
	if parent.Path == pagemap.RootPath {
		return placement{}, nil
	}
	if parent.WPID == nil {
		return placement{path: parent.Path, pending: true}, nil
	}
	return placement{path: parent.Path, wpID: *parent.WPID}, nil
}

func holdForParent(sc *run.StepContext, parent placement) run.Result {
	return run.Result{
		Next:   run.TransitionPause,
		Reason: run.PauseAwaitingParent,
		Message: sc.Page.Path + " waits for its parent " + parent.path +
			", which is not on the site yet; it goes on by itself once " + parent.path + " is",
	}
}
