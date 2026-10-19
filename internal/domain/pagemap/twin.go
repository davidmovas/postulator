package pagemap

func (i Index) HubTwin(term Page) (Page, bool) {
	if !term.WPType.Term() {
		return Page{}, false
	}
	var twin Page
	found := 0
	for at := range i.pages {
		candidate := i.pages[at]
		if !twinnable(candidate) || !sharesName(candidate, term) {
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
		if candidate.WPType.Term() && candidate.Status != StatusArchived && sharesName(candidate, page) {
			out = append(out, candidate)
		}
	}
	return out
}

func twinnable(page Page) bool {
	return page.WPType.core() && page.Status != StatusArchived
}

func sharesName(page, other Page) bool {
	for _, name := range names(other) {
		if carriesName(page, name) {
			return true
		}
	}
	return false
}

func names(page Page) []string {
	out := make([]string, 0, 4)
	for _, held := range []string{page.Observed.Title, page.Observed.H1, page.Title, page.H1} {
		if held != "" {
			out = append(out, held)
		}
	}
	return out
}
