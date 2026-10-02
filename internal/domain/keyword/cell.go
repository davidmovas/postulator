package keyword

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

const (
	separators = ",;|\n\r"
	thousand   = 1_000
	million    = 1_000_000
	largest    = 1_000_000_000_000
)

func closerOf(opener rune) (rune, bool) {
	switch opener {
	case '(':
		return ')', true
	case '[':
		return ']', true
	default:
		return 0, false
	}
}

func openerOf(closer rune) (rune, bool) {
	switch closer {
	case ')':
		return '(', true
	case ']':
		return '[', true
	default:
		return 0, false
	}
}

func Parse(cell string) (list List, unreadable []string) {
	fragments := split(cell)
	items := make([]Keyword, 0, len(fragments))
	for _, fragment := range fragments {
		item, readable := read(fragment)
		if !readable {
			unreadable = append(unreadable, fragment)
		}
		items = append(items, item)
	}
	return New(items), unreadable
}

func (l List) Cell() string {
	parts := make([]string, 0, len(l))
	for _, item := range l {
		if item.Volume == nil {
			parts = append(parts, item.Text)
			continue
		}
		parts = append(parts, item.Text+" ("+strconv.Itoa(*item.Volume)+")")
	}
	return strings.Join(parts, ", ")
}

func split(cell string) []string {
	runes := []rune(cell)
	var fragments []string
	start := 0
	for at := 0; at < len(runes); at++ {
		if end, bracketed := bracketEnd(runes, at); bracketed {
			at = end
			continue
		}
		if strings.ContainsRune(separators, runes[at]) {
			fragments = collect(fragments, runes[start:at])
			start = at + 1
		}
	}
	return collect(fragments, runes[start:])
}

func collect(fragments []string, runes []rune) []string {
	fragment := strings.TrimSpace(string(runes))
	if fragment == "" {
		return fragments
	}
	return append(fragments, fragment)
}

func bracketEnd(runes []rune, at int) (int, bool) {
	closer, opens := closerOf(runes[at])
	if !opens {
		return 0, false
	}
	for next := at + 1; next < len(runes); next++ {
		if runes[next] == closer {
			return next, true
		}
		if _, nested := closerOf(runes[next]); nested {
			return 0, false
		}
	}
	return 0, false
}

func read(fragment string) (Keyword, bool) {
	text, inside, closed, bracketed := tail([]rune(fragment))
	if !bracketed || !startsWithDigit(inside) {
		return Keyword{Text: fragment}, true
	}
	volume, readable := volumeOf(inside)
	switch {
	case !closed || !readable:
		return Keyword{Text: text}, false
	case text == "":
		return Keyword{}, false
	default:
		return Keyword{Text: text, Volume: new(volume)}, true
	}
}

func tail(runes []rune) (text, inside string, closed, bracketed bool) {
	if len(runes) == 0 {
		return "", "", false, false
	}
	last := len(runes) - 1
	if opener, closes := openerOf(runes[last]); closes {
		if at := lastIndex(runes[:last], opener); at >= 0 {
			return strings.TrimSpace(string(runes[:at])), strings.TrimSpace(string(runes[at+1 : last])), true, true
		}
	}
	for at := last; at >= 0; at-- {
		closer, opens := closerOf(runes[at])
		if !opens {
			continue
		}
		if lastIndex(runes[at+1:], closer) >= 0 {
			return "", "", false, false
		}
		return strings.TrimSpace(string(runes[:at])), strings.TrimSpace(string(runes[at+1:])), false, true
	}
	return "", "", false, false
}

func lastIndex(runes []rune, wanted rune) int {
	for at := len(runes) - 1; at >= 0; at-- {
		if runes[at] == wanted {
			return at
		}
	}
	return -1
}

func startsWithDigit(text string) bool {
	return text != "" && text[0] >= '0' && text[0] <= '9'
}

func volumeOf(raw string) (int, bool) {
	compact := strings.Map(withoutSpace, strings.ToLower(raw))
	switch {
	case strings.HasSuffix(compact, "k"):
		return scaled(strings.TrimSuffix(compact, "k"), thousand)
	case strings.HasSuffix(compact, "m"):
		return scaled(strings.TrimSuffix(compact, "m"), million)
	default:
		return grouped(compact)
	}
}

func withoutSpace(r rune) rune {
	if unicode.IsSpace(r) {
		return -1
	}
	return r
}

func grouped(compact string) (int, bool) {
	if digits(compact) {
		return whole(compact)
	}
	for _, mark := range []string{",", "."} {
		if !strings.Contains(compact, mark) {
			continue
		}
		groups := strings.Split(compact, mark)
		if !digits(groups[0]) || len(groups[0]) > 3 {
			return 0, false
		}
		for _, group := range groups[1:] {
			if len(group) != 3 || !digits(group) {
				return 0, false
			}
		}
		return whole(strings.Join(groups, ""))
	}
	return 0, false
}

func scaled(compact string, unit int) (int, bool) {
	units, fraction, fractional := strings.Cut(strings.Replace(compact, ",", ".", 1), ".")
	if !digits(units) || (fractional && !digits(fraction)) {
		return 0, false
	}
	amount, err := strconv.ParseFloat(units+"."+fraction+"0", 64)
	if err != nil {
		return 0, false
	}
	volume := math.Round(amount * float64(unit))
	if volume > largest {
		return 0, false
	}
	return int(volume), true
}

func whole(text string) (int, bool) {
	volume, err := strconv.Atoi(text)
	if err != nil || volume > largest {
		return 0, false
	}
	return volume, true
}

func digits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
