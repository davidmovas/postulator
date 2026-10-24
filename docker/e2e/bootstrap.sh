#!/bin/sh
set -eu

STAND="${STAND:-vanilla}"
MODE="${E2E_MODE:-test}"
PROJECT="${E2E_PROJECT:-postulator-test-vanilla}"
PROVISION="${E2E_PROVISION:-0}"
SITE_URL="${E2E_SITE_URL:-http://localhost:8088}"
WOO="${E2E_WOO:-1}"
PLUGIN_ZIP="/dist/postulator-companion.zip"
STAND_DIR="/e2e/stands/$STAND"
WP_ROOT="/var/www/html"
WP_IMAGE="${E2E_WP_IMAGE:-}"
WP_CLI_IMAGE="${E2E_WP_CLI_IMAGE:-}"

case "$STAND" in
	vanilla)
		TITLE="Vanilla Store"
		STAND_SEO="none"
		WOOCOMMERCE_VERSION="11.1.2"
		YOAST_VERSION="28.7"
		RANKMATH_VERSION="1.0.280"
		;;
	replica)
		TITLE="Peptide Store"
		STAND_SEO="rankmath"
		WOOCOMMERCE_VERSION="11.1.2"
		YOAST_VERSION="28.7"
		RANKMATH_VERSION="1.0.280"
		;;
	legacy)
		TITLE="Legacy Store"
		STAND_SEO="rankmath"
		WOOCOMMERCE_VERSION="6.5.2"
		YOAST_VERSION="18.3"
		RANKMATH_VERSION="1.0.98.1"
		;;
	*)
		echo "STAND must be vanilla, replica or legacy" >&2
		exit 1
		;;
esac
SEO="${E2E_SEO:-$STAND_SEO}"

case "$MODE" in
	test)
		ADMIN_USER="${E2E_ADMIN_USER:-shopadmin}"
		ADMIN_EMAIL="${E2E_ADMIN_EMAIL:-shopadmin@example.test}"
		ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-shop-admin-password}"
		PLUGIN="${E2E_PLUGIN:-1}"
		PROVISION="1"
		OUT="/e2e/.env.$STAND"
		;;
	sandbox)
		TITLE="Postulator E2E"
		ADMIN_USER="${E2E_ADMIN_USER:-postulator}"
		ADMIN_EMAIL="${E2E_ADMIN_EMAIL:-postulator@example.test}"
		ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-postulator-admin}"
		PLUGIN="${E2E_PLUGIN:-0}"
		if [ "$PROJECT" = "postulator-e2e" ]; then
			OUT="/e2e/.env.sandbox"
		else
			OUT="/e2e/.env.$PROJECT"
		fi
		;;
	*)
		echo "E2E_MODE must be test or sandbox" >&2
		exit 1
		;;
esac

for pair in "E2E_PLUGIN=$PLUGIN" "E2E_WOO=$WOO" "E2E_PROVISION=$PROVISION"; do
	case "${pair#*=}" in
		0 | 1)
			;;
		*)
			echo "${pair%%=*} must be 0 or 1" >&2
			exit 1
			;;
	esac
done

case "$SEO" in
	none | yoast | rankmath)
		;;
	*)
		echo "E2E_SEO must be none, yoast or rankmath" >&2
		exit 1
		;;
esac

if [ "$STAND" = "replica" ] && [ "$SEO" != "rankmath" ]; then
	echo "the replica stand runs Rank Math, so E2E_SEO must be rankmath or unset" >&2
	exit 1
fi
if [ "$STAND" = "replica" ] && [ "$PROVISION" = "1" ] && [ "$WOO" != "1" ]; then
	echo "the replica stand is a store, so E2E_WOO must be 1" >&2
	exit 1
fi
if [ "$STAND" = "legacy" ] && [ "$MODE" != "test" ]; then
	echo "the legacy stand pins plugins for the oldest WordPress the companion declares, so it runs only as a test stand" >&2
	exit 1
fi

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
		--title="$TITLE" \
		--admin_user="$ADMIN_USER" \
		--admin_password="$ADMIN_PASSWORD" \
		--admin_email="$ADMIN_EMAIL" \
		--skip-email
	fresh=1
	if [ "$MODE" = "test" ]; then
		wp db query "ALTER TABLE $(wp db prefix)terms AUTO_INCREMENT = 100000"
	fi
fi

install_companion() {
	if [ ! -f "$PLUGIN_ZIP" ]; then
		echo "$PLUGIN_ZIP is missing; run task plugin:zip first" >&2
		exit 1
	fi
	wp plugin install "$PLUGIN_ZIP" --force
}

ensure_plugin() {
	installed="$(wp plugin get "$1" --field=version 2>/dev/null || true)"
	if [ -z "$installed" ] || { [ "$MODE" = "test" ] && [ "$installed" != "$2" ]; }; then
		wp plugin install "$1" --version="$2" --force
	fi
	if ! wp plugin is-active "$1" >/dev/null 2>&1; then
		wp plugin activate "$1"
	fi
}

ensure_theme() {
	installed="$(wp theme get "$1" --field=version 2>/dev/null || true)"
	if [ -z "$installed" ] || { [ "$MODE" = "test" ] && [ "$installed" != "$2" ]; }; then
		wp theme install "$1" --version="$2" --force
	fi
	if ! wp theme is-active "$1" >/dev/null 2>&1; then
		wp theme activate "$1"
	fi
}

set_option() {
	if [ "$(wp option get "$1" 2>/dev/null || true)" != "$2" ]; then
		wp option update "$1" "$2"
	fi
}

page_id() {
	wp post list --post_type=page --post_status=publish,draft,pending,private,future \
		--name="$1" --post_parent="$2" --field=ID
}

ensure_page() {
	found="$(page_id "$1" "$3")"
	if [ -n "$found" ]; then
		echo "$found"
		return
	fi
	wp post create \
		--user="$ADMIN_USER" \
		--post_type=page \
		--post_status=publish \
		--post_name="$1" \
		--post_title="$2" \
		--post_parent="$3" \
		--post_content="$4" \
		--porcelain
}

ensure_term_description() {
	term_id="$(wp term list "$1" --slug="$2" --field=term_id)"
	if [ -z "$term_id" ]; then
		echo "the $1 term $2 is missing" >&2
		exit 1
	fi
	if [ "$(wp term get "$1" "$term_id" --field=description)" != "$3" ]; then
		wp term update "$1" "$term_id" --description="$3"
	fi
}

import_products() {
	wp eval-file /e2e/import-products.php "$1" --user="$ADMIN_USER"
}

set_permalinks() {
	if [ "$(wp option get permalink_structure)" != "/%postname%/" ]; then
		wp rewrite structure '/%postname%/' --hard
	fi
	wp rewrite flush --hard
}

sandbox_password() {
	if ! wp user application-password list "$ADMIN_USER" --field=name | grep -qx postulator-sandbox; then
		APP_PASSWORD="$(wp user application-password create "$ADMIN_USER" postulator-sandbox --porcelain)"
		{
			echo "SANDBOX_WP_URL=$SITE_URL"
			echo "SANDBOX_WP_USER=$ADMIN_USER"
			echo "SANDBOX_WP_APP_PASSWORD=$APP_PASSWORD"
		} > "$OUT"
	fi
}

if [ "$MODE" = "test" ]; then
	install_companion
	if [ "$PLUGIN" = "1" ]; then
		wp plugin activate postulator-companion
	elif wp plugin is-active postulator-companion >/dev/null 2>&1; then
		wp plugin deactivate postulator-companion
	fi
elif [ "$PLUGIN" = "1" ]; then
	install_companion
	wp plugin activate postulator-companion
fi

if [ "$PROVISION" = "0" ]; then
	if [ "$fresh" = "1" ]; then
		set_permalinks
	fi
	sandbox_password
	echo "the sandbox is ready at $SITE_URL"
	exit 0
fi

for slug in wordpress-seo seo-by-rank-math; do
	case "$SEO:$slug" in
		yoast:wordpress-seo | rankmath:seo-by-rank-math)
			continue
			;;
	esac
	if wp plugin is-active "$slug" >/dev/null 2>&1; then
		wp plugin deactivate "$slug"
	fi
done

case "$SEO" in
	yoast)
		ensure_plugin wordpress-seo "$YOAST_VERSION"
		wp yoast index --reindex
		;;
	rankmath)
		ensure_plugin seo-by-rank-math "$RANKMATH_VERSION"
		set_option rank_math_is_configured 1
		set_option rank_math_registration_skip 1
		;;
esac

if [ "$WOO" = "1" ]; then
	ensure_plugin woocommerce "$WOOCOMMERCE_VERSION"
	set_option woocommerce_coming_soon no
fi

. "$STAND_DIR/provision.sh"

set_permalinks

if [ "$MODE" = "sandbox" ]; then
	sandbox_password
	echo "the sandbox is provisioned as the $STAND stand at $SITE_URL"
	exit 0
fi

if [ "$(wp user application-password list "$ADMIN_USER" --format=count)" != "0" ]; then
	wp user application-password delete "$ADMIN_USER" --all
fi
APP_PASSWORD="$(wp user application-password create "$ADMIN_USER" e2e --porcelain)"
RESTRICTED_PASSWORD=""
if [ -n "${RESTRICTED_USER:-}" ]; then
	if [ "$(wp user application-password list "$RESTRICTED_USER" --format=count)" != "0" ]; then
		wp user application-password delete "$RESTRICTED_USER" --all
	fi
	RESTRICTED_PASSWORD="$(wp user application-password create "$RESTRICTED_USER" e2e --porcelain)"
fi

{
	echo "E2E_STAND=$STAND"
	echo "E2E_PROJECT=$PROJECT"
	echo "E2E_WP_URL=$SITE_URL"
	echo "E2E_WP_USER=$ADMIN_USER"
	echo "E2E_WP_APP_PASSWORD=$APP_PASSWORD"
	echo "E2E_SEO=$SEO"
	echo "E2E_WOO=$WOO"
	echo "E2E_PLUGIN=$PLUGIN"
	echo "E2E_WP_IMAGE=$WP_IMAGE"
	echo "E2E_WP_CLI_IMAGE=$WP_CLI_IMAGE"
	if [ -n "$RESTRICTED_PASSWORD" ]; then
		echo "E2E_RESTRICTED_USER=$RESTRICTED_USER"
		echo "E2E_RESTRICTED_APP_PASSWORD=$RESTRICTED_PASSWORD"
	fi
} > "$OUT"

echo "the $STAND stand is provisioned at $SITE_URL on WordPress $(wp core version), $WP_IMAGE and $WP_CLI_IMAGE, with the active plugins:"
wp plugin list --status=active --fields=name,version --format=csv
