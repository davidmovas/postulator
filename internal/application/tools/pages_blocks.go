package tools

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagesBlocksName = "pages_blocks"
	opListBlocks    = "list"
	allBlocks       = 500
)

type pagesBlocksArgs struct {
	Op       string `json:"op" enum:"list,replace,insert_after,delete,place_link" description:"All but list change the site"`
	PageID   string `json:"pageId,omitempty" description:"Page id, or give path"`
	Path     string `json:"path,omitempty" description:"Page path"`
	Index    *int   `json:"index,omitempty" description:"Block index from list; insert_after -1 is the start"`
	Hash     string `json:"hash,omitempty" description:"bodyHash from list"`
	HTML     string `json:"html,omitempty" description:"The block's html; links only by place_link"`
	TargetID string `json:"targetId,omitempty" description:"place_link: page id"`
	Anchor   string `json:"anchor,omitempty" description:"place_link: text in the block"`
	Text     bool   `json:"text,omitempty" description:"list: each block's whole text and links, a long block in parts"`
	Cursor   string `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit    int    `json:"limit,omitempty" description:"list: default 50"`
}

type PagesBlocksResponse struct {
	Blocks *sitemodel.BlocksView `json:"blocks,omitempty"`
	Edit   *runs.EditResponse    `json:"edit,omitempty"`
}

func pagesBlocks(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name: pagesBlocksName,
		Description: "Read a page's blocks with their owners, or change one on the site as a run the person confirms and " +
			"runs_revert undoes; the owner's blocks stay unless the page type rewrites.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in pagesBlocksArgs) (PagesBlocksResponse, error) {
		if in.Op == opListBlocks {
			read, err := fitted(deps, in.Limit, func(limit int) (sitemodel.BlocksView, error) {
				return deps.SiteModel.Blocks(ctx, sitemodel.BlocksRequest{
					SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Cursor: in.Cursor, Limit: limit, Text: in.Text,
				})
			})
			return PagesBlocksResponse{Blocks: &read}, err
		}
		read, err := deps.SiteModel.Blocks(ctx, sitemodel.BlocksRequest{
			SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Limit: allBlocks,
		})
		if err != nil {
			return PagesBlocksResponse{}, err
		}
		started, err := deps.Runs.StartEdit(ctx, runs.EditRequest{SiteID: b.SiteID, PageID: read.PageID, Edit: in.edit()})
		if err != nil {
			return PagesBlocksResponse{}, err
		}
		return PagesBlocksResponse{Edit: &started}, nil
	}), func(ctx context.Context, b Binding, in pagesBlocksArgs) (string, error) {
		if in.Op == opListBlocks {
			return "", nil
		}
		return deps.blockSummary(ctx, b, in)
	})
	return readsOn(tool, opListBlocks)
}

func (in *pagesBlocksArgs) edit() run.Edit {
	index := 0
	if in.Index != nil {
		index = *in.Index
	}
	return run.Edit{Op: run.EditOp(in.Op), Index: index, Hash: in.Hash, HTML: in.HTML, TargetID: in.TargetID, Anchor: in.Anchor}
}

func refusedBlock(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func (deps Deps) blockSummary(ctx context.Context, b Binding, in pagesBlocksArgs) (string, error) {
	if in.Index == nil {
		return "", refusedBlock("name the block by the index list answered", "index")
	}
	wanted := in.edit()
	if err := wanted.Validate(); err != nil {
		return "", err
	}
	read, err := deps.SiteModel.Blocks(ctx, sitemodel.BlocksRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Limit: allBlocks})
	if err != nil {
		return "", err
	}
	if refused := deps.editable(ctx, b, read, wanted); refused != nil {
		return "", refused
	}
	blocks := read.Blocks.Items
	others := strconv.Itoa(len(blocks) - 1)
	switch wanted.Op {
	case run.EditReplace:
		return "Replace block " + strconv.Itoa(wanted.Index) + " of " + read.Path + " (" + described(blocks[wanted.Index]) +
			") with " + quoted(block.Parse(wanted.HTML).Text()) + "; the other " + others + " blocks stay as they are", nil
	case run.EditDelete:
		return "Delete block " + strconv.Itoa(wanted.Index) + " of " + read.Path + " (" + described(blocks[wanted.Index]) +
			"); the other " + others + " blocks stay as they are", nil
	case run.EditInsertAfter:
		where := "at the start of " + read.Path
		if wanted.Index >= 0 {
			where = "after block " + strconv.Itoa(wanted.Index) + " of " + read.Path + " (" + described(blocks[wanted.Index]) + ")"
		}
		return "Insert a block " + where + ": " + quoted(block.Parse(wanted.HTML).Text()) + "; the " +
			strconv.Itoa(len(blocks)) + " blocks there stay as they are", nil
	default:
		target, pathErr := deps.pathOf(ctx, wanted.TargetID)
		if pathErr != nil {
			return "", pathErr
		}
		return "Turn " + quoted(wanted.Anchor) + " in block " + strconv.Itoa(wanted.Index) + " of " + read.Path + " (" +
			described(blocks[wanted.Index]) + ") into a link to " + target + "; nothing else in the block changes", nil
	}
}

func (deps Deps) editable(ctx context.Context, b Binding, read sitemodel.BlocksView, wanted run.Edit) error {
	if !read.OnSite {
		return refusedBlock("the page "+read.Path+" is not on the site, so there is no body to edit there", "pageId")
	}
	owner, err := deps.Sites.Get(ctx, sites.GetRequest{ID: b.SiteID})
	if err != nil {
		return err
	}
	if !owner.Site.Plugin.Installed {
		return refusedBlock("a block is edited through the companion plugin, which this site does not have", "pageId")
	}
	if read.BodyHash != wanted.Hash {
		return errors.New(errors.Conflict, "the body of "+read.Path+" changed since you listed its blocks; list them again").
			WithDetail("field", "hash")
	}
	blocks := read.Blocks.Items
	if wanted.Index >= len(blocks) {
		return refusedBlock("the page holds "+strconv.Itoa(len(blocks))+" blocks, so it has no block "+strconv.Itoa(wanted.Index), "index")
	}
	if (wanted.Op == run.EditReplace || wanted.Op == run.EditDelete) && blocks[wanted.Index].Owner != sitemodel.OwnerOurs && !read.Rewrites {
		return refusedBlock("block "+strconv.Itoa(wanted.Index)+" is the owner's own, and the page type's body keeps the owner's "+
			"text; insert a block beside it, or switch rewrites on for the type first", "index")
	}
	if wanted.Op == run.EditPlaceLink {
		target, err := deps.Pages.Get(ctx, pages.GetRequest{ID: wanted.TargetID})
		if err != nil {
			return err
		}
		if target.Page.SiteID != b.SiteID || target.Page.WPID == nil {
			return refusedBlock("the page the link points to is not on this site", "targetId")
		}
	}
	return nil
}

func described(row sitemodel.BlockRow) string {
	owner := "the owner's"
	if row.Owner == sitemodel.OwnerOurs {
		owner = "ours"
	}
	return owner + ", a " + row.Kind + ": " + quoted(row.Excerpt)
}
