<?php
/**
 * Plugin Name: Postulator Companion
 * Version: 1.2.0
 * Requires at least: 6.4
 * Requires PHP: 8.1
 */

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const VERSION             = '1.2.0';
const NAMESPACE_PATH      = 'postulator/v1';
const CAPABILITIES        = array( 'bulk', 'seo_meta', 'seo_meta_read', 'content_hash', 'raw', 'preview' );
const TYPES               = array( 'page', 'post', 'product', 'product_cat' );
const TERM_TYPES          = array( 'product_cat' );
const DEFAULT_LIMIT       = 100;
const MAX_LIMIT           = 500;
const TERM_MODIFIED_KEY   = '_postulator_modified';
const PREVIEW_TTL         = HOUR_IN_SECONDS;
const PREVIEW_QUERY_VAR   = 'postulator_preview';
const PREVIEW_HASH_KEY    = '_postulator_preview_hash';
const PREVIEW_EXPIRES_KEY = '_postulator_preview_expires';
const PREVIEW_STATUSES    = array( 'draft', 'pending', 'future', 'private' );

require_once __DIR__ . '/includes/http.php';
require_once __DIR__ . '/includes/normalize.php';
require_once __DIR__ . '/includes/seo.php';
require_once __DIR__ . '/includes/content.php';
require_once __DIR__ . '/includes/head.php';
require_once __DIR__ . '/includes/preview.php';
require_once __DIR__ . '/includes/routes.php';

add_action( 'rest_api_init', __NAMESPACE__ . '\\register_routes' );
add_action( 'init', __NAMESPACE__ . '\\boot_head', 20 );
add_filter( 'query_vars', __NAMESPACE__ . '\\preview_query_vars' );
add_action( 'pre_get_posts', __NAMESPACE__ . '\\preview_query' );
add_action( 'created_term', __NAMESPACE__ . '\\touch_term', 10, 3 );
add_action( 'edited_term', __NAMESPACE__ . '\\touch_term', 10, 3 );
