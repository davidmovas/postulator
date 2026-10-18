HOME_BODY='Research peptides for the laboratory, shipped cold and sealed under nitrogen.

Browse the <a href="/peptides/">peptides</a> or read about <a href="/bpc-157/">BPC-157</a> and <a href="/tb-500/">TB-500</a>.'

TB_BODY='TB-500 is a synthetic fragment of thymosin beta-4, sold for research use only.

Every vial ships with a certificate of analysis. Compare it with <a href="/bpc-157/">BPC-157</a> before you order.'

SHORTCODE_BODY='[products category="healing-peptides" limit="4" columns="2"]'

GUIDES_BODY='Step-by-step guides for handling research peptides in the laboratory.'

RECONSTITUTION_BODY='Reconstitute a lyophilized peptide with bacteriostatic water, adding it slowly along the wall of the vial.

Swirl gently and never shake the vial.'

MU_PLUGINS="$WP_ROOT/wp-content/mu-plugins"

ensure_plugin classic-editor 1.7.0
ensure_theme blocksy 2.1.60

mkdir -p "$MU_PLUGINS"
if ! cmp -s "$STAND_DIR/site-behaviour.php" "$MU_PLUGINS/site-behaviour.php"; then
	cp "$STAND_DIR/site-behaviour.php" "$MU_PLUGINS/site-behaviour.php"
fi

if [ "$(wp option pluck rank-math-options-general breadcrumbs)" != "on" ]; then
	wp option patch update rank-math-options-general breadcrumbs on
fi

import_products "$STAND_DIR/products.csv"
ensure_term_description product_cat peptides "Research peptides for the laboratory."

home_id="$(ensure_page home Home 0 "$HOME_BODY")"
ensure_page peptides Peptides 0 ""
ensure_page bpc-157 BPC-157 0 "$SHORTCODE_BODY"
ensure_page tb-500 TB-500 0 "$TB_BODY"
ensure_page healing-peptides "Healing Peptides" 0 ""
ensure_page lab-guides "Lab Guides" 0 "$GUIDES_BODY"
ensure_page reconstitution Reconstitution 0 "$RECONSTITUTION_BODY"

set_option show_on_front page
set_option page_on_front "$home_id"
