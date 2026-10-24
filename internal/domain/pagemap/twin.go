package pagemap

func (i Index) HubTwin(term Page) (Page, bool) {
	if !term.WPType.Term() {
		return Page{}, false
	}
	var twin Page
	found := 0
	for at := range i.pages {
		candidate := i.pages[at]
		if !twinnable(candidate) || !i.sharesName(candidate, term) {
			continue
		}
		twin = candidate
		found++
	}
	if found != 1 {
		return Page{}, false
	}
	return twin, true
}

func (i Index) TermTwins(page Page) []Page {
	out := make([]Page, 0)
	if !twinnable(page) {
		return out
	}
	for at := range i.pages {
		candidate := i.pages[at]
		if candidate.WPType.Term() && candidate.Status != StatusArchived && i.sharesName(candidate, page) {
			out = append(out, candidate)
		}
	}
	return out
}

func twinnable(page Page) bool {
	return page.WPType == WPPage && page.Status != StatusArchived
}

func (i Index) sharesName(page, other Page) bool {
	held := i.namesOf(page)
	for _, name := range i.namesOf(other) {
		key := nameKey(name)
		for _, own := range held {
			if key != "" && nameKey(own) == key {
				return true
			}
		}
	}
	return false
}

func (i Index) namesOf(page Page) []string {
	known := i.names[page.ID]
	out := make([]string, 0, len(known)+2)
	for _, held := range append([]string{page.Title, page.H1}, known...) {
		if held != "" {
			out = append(out, held)
		}
	}
	return out
}
