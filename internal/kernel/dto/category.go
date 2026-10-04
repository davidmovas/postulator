package dto

type Category struct {
	ID     string `json:"id" description:"The id of the category"`
	Name   string `json:"name" description:"The category's name"`
	TermID *int64 `json:"termId,omitempty" minimum:"1" description:"The WordPress term id, left out until the site has the term"`
}
