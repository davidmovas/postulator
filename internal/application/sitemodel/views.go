package sitemodel

import "github.com/davidmovas/postulator/internal/domain/contextpack"

type Section struct {
	Name     string   `json:"name"`
	Heading  string   `json:"heading"`
	Priority int      `json:"priority"`
	Lines    []string `json:"lines"`
	Items    []string `json:"items"`
	Tokens   int      `json:"tokens"`
}

type Drop struct {
	Section string `json:"section"`
	Heading string `json:"heading"`
	Items   int    `json:"items"`
	Tokens  int    `json:"tokens"`
	Whole   bool   `json:"whole"`
}

type PackView struct {
	PageID   string    `json:"pageId"`
	Path     string    `json:"path"`
	Field    string    `json:"field"`
	Budget   int       `json:"budget"`
	Built    int       `json:"built"`
	Tokens   int       `json:"tokens"`
	Sections []Section `json:"sections"`
	Dropped  []Drop    `json:"dropped"`
}

func packView(picture Picture) PackView {
	view := PackView{
		PageID: picture.Facts.Page.PageID, Path: picture.Facts.Page.Path, Field: string(picture.Fitted.Field),
		Budget: picture.Fitted.Budget, Built: picture.Pack.Tokens, Tokens: picture.Fitted.Tokens,
		Sections: make([]Section, 0, len(picture.Fitted.Sections)), Dropped: make([]Drop, 0, len(picture.Fitted.Dropped)),
	}
	for _, section := range picture.Fitted.Sections {
		view.Sections = append(view.Sections, sectionView(section))
	}
	for _, drop := range picture.Fitted.Dropped {
		view.Dropped = append(view.Dropped, Drop{Section: string(drop.Section), Heading: drop.Heading, Items: drop.Items, Tokens: drop.Tokens, Whole: drop.Whole})
	}
	return view
}

func sectionView(section contextpack.Section) Section {
	return Section{
		Name: string(section.Name), Heading: section.Heading, Priority: section.Priority,
		Lines: nonNil(section.Lines), Items: nonNil(section.Items), Tokens: section.Tokens,
	}
}

func nonNil(lines []string) []string {
	if lines == nil {
		return []string{}
	}
	return lines
}
