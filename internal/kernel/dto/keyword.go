package dto

type Keyword struct {
	Text   string `json:"text" description:"Phrase"`
	Volume *int   `json:"volume,omitempty" minimum:"0" description:"Monthly searches, if known"`
}
