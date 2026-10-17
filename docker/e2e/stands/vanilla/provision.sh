GUIDE_BODY='<!-- wp:paragraph -->
<p>Every order ships with a short guide to storing and brewing what you bought.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Start with the <a href="/guide/setup/">setup</a> before the first cup.</p>
<!-- /wp:paragraph -->'

SETUP_BODY='<!-- wp:heading -->
<h2 class="wp-block-heading">Grinding</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Grind the beans just before you brew them, and keep the rest sealed.</p>
<!-- /wp:paragraph -->'

ensure_theme twentytwentyfive 1.5
set_option show_on_front posts

guide_id="$(ensure_page guide Guide 0 "$GUIDE_BODY")"
ensure_page setup Setup "$guide_id" "$SETUP_BODY"

if [ "$WOO" = "1" ]; then
	import_products "$STAND_DIR/products.csv"
	ensure_term_description product_cat koffein "Caffeine products."
fi
