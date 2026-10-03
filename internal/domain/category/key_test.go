package category_test

import (
	"html"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/category"
)

func sameTermName(left, right string) bool {
	plain := func(name string) string {
		return strings.Join(strings.Fields(html.UnescapeString(name)), " ")
	}
	plainLeft, plainRight := plain(left), plain(right)
	return plainLeft != "" && strings.EqualFold(plainLeft, plainRight)
}

func TestKeyNormalisesAName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain ascii is lowered", in: "Healing", want: "healing"},
		{name: "the ends are trimmed", in: "  Healing \t", want: "healing"},
		{name: "inner whitespace collapses to one space", in: "Healing \t\n  Peptides", want: "healing peptides"},
		{name: "an html entity is decoded", in: "Tools &amp; Kits", want: "tools & kits"},
		{name: "a numeric entity is decoded", in: "Tools &#038; Kits", want: "tools & kits"},
		{name: "a non-breaking space is whitespace", in: "Healing&nbsp;Peptides", want: "healing peptides"},
		{name: "cyrillic is lowered", in: "ПЕПТИДЫ", want: "пептиды"},
		{name: "a capital sharp s folds to the small one", in: "STRAẞE", want: "straße"},
		{name: "a sharp s is not two letters", in: "Straße", want: "straße"},
		{name: "a final sigma folds with the medial one", in: "ΟΔΟΣ οδος", want: "οδοσ οδοσ"},
		{name: "the kelvin sign folds to k", in: "Kits", want: "kits"},
		{name: "a dotted capital i stays itself", in: "İzmir", want: "İzmir"},
		{name: "a blank name has an empty key", in: " \t ", want: ""},
		{name: "an entity of a space alone is blank", in: "&nbsp;", want: ""},
		{name: "an empty name has an empty key", in: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := category.Key(tc.in); got != tc.want {
				t.Fatalf("Key(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestKeyAgreesWithTheWordPressTermNameRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		left, right string
		same        bool
	}{
		{name: "the same name", left: "Healing", right: "Healing", same: true},
		{name: "a different case", left: "healing", right: "HEALING", same: true},
		{name: "padding and inner runs of space", left: "  Healing   Peptides ", right: "Healing Peptides", same: true},
		{name: "an escaped ampersand", left: "Tools &amp; Kits", right: "tools & kits", same: true},
		{name: "a non-breaking space", left: "Healing&nbsp;Peptides", right: "healing peptides", same: true},
		{name: "a capital sharp s", left: "STRAẞE", right: "straße", same: true},
		{name: "a sharp s against a double s", left: "Straße", right: "Strasse"},
		{name: "a final sigma", left: "ΟΔΟΣ", right: "οδος", same: true},
		{name: "a long s", left: "ſtrength", right: "Strength", same: true},
		{name: "the kelvin sign", left: "Kits", right: "KITS", same: true},
		{name: "a dotted capital i against a plain i", left: "İzmir", right: "izmir"},
		{name: "a dotless i against a plain i", left: "ızmir", right: "izmir"},
		{name: "a hyphen is not a space", left: "BPC-157", right: "BPC 157"},
		{name: "a hyphen against a space-less name", left: "BPC-157", right: "BPC157"},
		{name: "two different names", left: "Healing", right: "Recovery"},
		{name: "an escape that is not decoded twice", left: "A &amp;amp; B", right: "A & B"},
		{name: "a double escape decoded once on both sides", left: "A &amp;amp; B", right: "a &amp;amp; b", same: true},
		{name: "a double escape against a single one", left: "A &amp;amp; B", right: "a &amp; b"},
		{name: "two invalid bytes read as the same replacement", left: "\xff", right: "\xfe", same: true},
		{name: "an invalid byte and the replacement character", left: "\xff", right: "�", same: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if rule := sameTermName(tc.left, tc.right); rule != tc.same {
				t.Fatalf("the WordPress rule says %q and %q are the same: %t, the table says %t", tc.left, tc.right, rule, tc.same)
			}
			left, right := category.Key(tc.left), category.Key(tc.right)
			if got := left == right; got != tc.same {
				t.Fatalf("Key(%q) = %q, Key(%q) = %q; equal %t, want %t", tc.left, left, tc.right, right, got, tc.same)
			}
		})
	}
}

func TestKeyFoldsEveryRuneToOneMemberOfItsCaseOrbit(t *testing.T) {
	t.Parallel()

	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !utf8.ValidRune(r) || unicode.SimpleFold(r) == r {
			continue
		}
		key := category.Key(string(r))
		if !strings.EqualFold(key, string(r)) || utf8.RuneCountInString(key) != 1 {
			t.Fatalf("Key(%U) = %q, which is not a case of it", r, key)
		}
		for at := unicode.SimpleFold(r); at != r; at = unicode.SimpleFold(at) {
			if other := category.Key(string(at)); other != key {
				t.Fatalf("Key(%U) = %q but Key(%U) = %q, though they fold together", r, key, at, other)
			}
		}
	}
}

func TestKeyIsLowercaseWhereTheOrbitHasALowercase(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"K", "K", "S", "ſ", "Σ", "ς", "Ǆ", "ǅ", "ẞ"} {
		key := category.Key(in)
		r, _ := utf8.DecodeRuneInString(key)
		if !unicode.IsLower(r) {
			t.Errorf("Key(%q) = %q, want the lowercase member of its orbit", in, key)
		}
	}
}
