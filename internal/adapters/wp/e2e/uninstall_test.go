//go:build e2e

package e2e_test

import (
	"fmt"
	"testing"
)

const bookkeepingCount = `global $wpdb; echo (int) $wpdb->get_var("SELECT COUNT(*) FROM {$wpdb->postmeta} WHERE meta_key IN ('_postulator_preview_hash', '_postulator_preview_expires')") + (int) $wpdb->get_var("SELECT COUNT(*) FROM {$wpdb->termmeta} WHERE meta_key = '_postulator_modified'");`

func seoTitleKey(seoPlugin string) string {
	switch seoPlugin {
	case "rankmath":
		return "rank_math_title"
	case "yoast":
		return "_yoast_wpseo_title"
	default:
		return "_postulator_seo_title"
	}
}

func TestUninstallRemovesTheBookkeepingAndKeepsTheSEOMeta(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)
	t.Cleanup(func() {
		wpCLI(t, env, "plugin", "install", "/dist/postulator-companion.zip", "--force", "--activate")
	})

	draft := createPage(t, c, pageSpec{title: "Preview kept", slug: uniqueSlug("kept"), content: "<p>x</p>", status: "draft"})
	issuePreview(t, c, draft)
	listContent(t, c, "types=category&limit=500")
	page := createPage(t, c, pageSpec{title: "Titled", slug: uniqueSlug("titled"), content: "<p>x</p>"})
	writeSEO(t, c, page, map[string]string{"title": "Shop SEO title kept"})

	if before := wpCLI(t, env, "eval", bookkeepingCount); before == "0" {
		t.Fatal("the plugin stored no bookkeeping, so the uninstall would prove nothing")
	}

	wpCLI(t, env, "plugin", "uninstall", "postulator-companion", "--deactivate")

	if after := wpCLI(t, env, "eval", bookkeepingCount); after != "0" {
		t.Errorf("after the uninstall %s rows of the plugin's bookkeeping remain", after)
	}
	key := seoTitleKey(stand.seoPlugin)
	if kept := wpCLI(t, env, "eval", fmt.Sprintf("echo get_post_meta(%d, '%s', true);", page, key)); kept != "Shop SEO title kept" {
		t.Errorf("after the uninstall %s = %q, want the SEO title the client keeps", key, kept)
	}
}
