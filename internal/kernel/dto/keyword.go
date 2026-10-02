package dto

type Keyword struct {
	Text   string `json:"text" description:"The phrase"`
	Volume *int   `json:"volume,omitempty" minimum:"0" description:"Monthly searches, left out when unknown"`
}
