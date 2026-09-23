#!/bin/sh
set -eu

MODE="${E2E_MODE:-sandbox}"
SITE_URL="${E2E_SITE_URL:-http://localhost:8089}"
ADMIN_USER="${E2E_ADMIN_USER:-postulator}"
ADMIN_EMAIL="${E2E_ADMIN_EMAIL:-postulator@example.test}"
ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-postulator-admin}"
SEO="${E2E_SEO:-none}"
WOO="${E2E_WOO:-1}"
PLUGIN="${E2E_PLUGIN:-1}"
PLUGIN_ZIP="/dist/postulator-companion.zip"

case "$MODE" in
	sandbox)
		OUT="/e2e/.env.sandbox"
		;;
	test)
		OUT="/e2e/.env.generated"
		;;
	*)
		echo "E2E_MODE must be sandbox or test" >&2
		exit 1
		;;
esac

attempt=0
until wp db check >/dev/null 2>&1; do
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 60 ]; then
		echo "the database did not become reachable" >&2
		exit 1
	fi
	sleep 2
done

fresh=0
if ! wp core is-installed >/dev/null 2>&1; then
	wp core install \
		--url="$SITE_URL" \
		--title="Postulator E2E" \
		--admin_user="$ADMIN_USER" \
		--admin_password="$ADMIN_PASSWORD" \
		--admin_email="$ADMIN_EMAIL" \
		--skip-email
	fresh=1
fi

if [ ! -f "$PLUGIN_ZIP" ]; then
	echo "$PLUGIN_ZIP is missing; run task plugin:zip first" >&2
	exit 1
fi
wp plugin install "$PLUGIN_ZIP" --force

if [ "$MODE" = "sandbox" ]; then
	wp plugin activate postulator-companion
	if [ "$fresh" = "1" ]; then
		wp rewrite structure '/%postname%/' --hard
		wp rewrite flush --hard
	fi
	if ! wp user application-password list "$ADMIN_USER" --field=name | grep -qx postulator-sandbox; then
		APP_PASSWORD="$(wp user application-password create "$ADMIN_USER" postulator-sandbox --porcelain)"
		{
			echo "SANDBOX_WP_URL=$SITE_URL"
			echo "SANDBOX_WP_USER=$ADMIN_USER"
			echo "SANDBOX_WP_APP_PASSWORD=$APP_PASSWORD"
		} > "$OUT"
	fi
	echo "the sandbox is ready at $SITE_URL"
	exit 0
fi

case "$PLUGIN" in
	1)
		wp plugin activate postulator-companion
		;;
	0)
		if wp plugin is-active postulator-companion >/dev/null 2>&1; then
			wp plugin deactivate postulator-companion
		fi
		;;
	*)
		echo "E2E_PLUGIN must be 0 or 1" >&2
		exit 1
		;;
esac

for slug in wordpress-seo seo-by-rank-math; do
	if wp plugin is-active "$slug" >/dev/null 2>&1; then
		wp plugin deactivate "$slug"
	fi
done

case "$SEO" in
	none)
		;;
	yoast)
		wp plugin install wordpress-seo --activate
		wp yoast index --reindex
		;;
	rankmath)
		wp plugin install seo-by-rank-math --activate
		wp option update rank_math_is_configured 1
		wp option update rank_math_registration_skip 1
		;;
	*)
		echo "E2E_SEO must be none, yoast or rankmath" >&2
		exit 1
		;;
esac

if [ "$WOO" = "1" ] && ! wp plugin is-active woocommerce >/dev/null 2>&1; then
	wp plugin install woocommerce --activate
fi

if [ "$WOO" = "1" ]; then
	if [ "$(wp term list product_cat --slug=postulator-koffein --format=count)" = "0" ]; then
		wp term create product_cat "Koffein" \
			--slug=postulator-koffein \
			--description="Caffeine products."
	fi
	if [ "$(wp post list --post_type=product --name=postulator-powder --format=count)" = "0" ]; then
		PRODUCT_ID="$(wp post create \
			--post_type=product \
			--post_title="Powder" \
			--post_name=postulator-powder \
			--post_status=publish \
			--post_content='<p>See <a href="/product-category/postulator-koffein/">Koffein</a>.</p>' \
			--porcelain)"
		wp post term add "$PRODUCT_ID" product_cat postulator-koffein
	fi
fi

wp rewrite structure '/%postname%/' --hard
wp rewrite flush --hard

if [ "$(wp user application-password list "$ADMIN_USER" --format=count)" != "0" ]; then
	wp user application-password delete "$ADMIN_USER" --all
fi
APP_PASSWORD="$(wp user application-password create "$ADMIN_USER" postulator-e2e --porcelain)"

{
	echo "E2E_WP_URL=$SITE_URL"
	echo "E2E_WP_USER=$ADMIN_USER"
	echo "E2E_WP_APP_PASSWORD=$APP_PASSWORD"
	echo "E2E_SEO=$SEO"
	echo "E2E_WOO=$WOO"
	echo "E2E_PLUGIN=$PLUGIN"
} > "$OUT"

echo "wordpress is provisioned at $SITE_URL"
