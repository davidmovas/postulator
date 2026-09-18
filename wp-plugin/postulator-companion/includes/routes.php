<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function register_routes(): void {
	register_rest_route(
		NAMESPACE_PATH,
		'/manifest',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\manifest',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);
}

function manifest(): \WP_REST_Response {
	return new \WP_REST_Response(
		array(
			'version'      => VERSION,
			'capabilities' => CAPABILITIES,
			'seoPlugin'    => detect_plugin(),
			'wpVersion'    => (string) get_bloginfo( 'version' ),
			'site'         => (string) home_url(),
		),
		200
	);
}
