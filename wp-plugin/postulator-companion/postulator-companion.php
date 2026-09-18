<?php
/**
 * Plugin Name: Postulator Companion
 * Version: 1.0.0
 * Requires at least: 6.4
 * Requires PHP: 8.1
 */

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const VERSION           = '1.0.0';
const NAMESPACE_PATH    = 'postulator/v1';
const CAPABILITIES      = array( 'bulk', 'seo_meta', 'content_hash', 'raw' );
const TYPES             = array( 'page', 'post', 'product' );
const TERM_TYPES        = array( 'product_cat' );
const DEFAULT_LIMIT     = 100;
const MAX_LIMIT         = 500;
const TERM_MODIFIED_KEY = '_postulator_modified';
const INSTALLED_OPTION  = 'postulator_companion_installed_at';

require_once __DIR__ . '/includes/http.php';
require_once __DIR__ . '/includes/normalize.php';
require_once __DIR__ . '/includes/seo.php';
require_once __DIR__ . '/includes/content.php';
require_once __DIR__ . '/includes/routes.php';

register_activation_hook( __FILE__, __NAMESPACE__ . '\\activate' );

add_action( 'rest_api_init', __NAMESPACE__ . '\\register_routes' );
add_action( 'created_term', __NAMESPACE__ . '\\touch_term', 10, 3 );
add_action( 'edited_term', __NAMESPACE__ . '\\touch_term', 10, 3 );

function activate(): void {
	add_option( INSTALLED_OPTION, gmdate( 'Y-m-d\TH:i:s\Z' ) );
}
