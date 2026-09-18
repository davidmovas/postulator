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

	register_rest_route(
		NAMESPACE_PATH,
		'/content',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\content_list',
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

function parse_types( string $types ) {
	if ( '' === $types ) {
		return TYPES;
	}

	$requested = array_values( array_filter( array_map( 'trim', explode( ',', $types ) ), 'strlen' ) );
	foreach ( $requested as $type ) {
		if ( ! in_array( $type, TYPES, true ) ) {
			return invalid( 'invalid_type', 'unknown content type: ' . $type );
		}
	}
	return array_values( array_unique( $requested ) );
}

function parse_since( string $since ) {
	if ( '' === $since ) {
		return '';
	}

	if ( 1 !== preg_match( '/^\d{4}-\d{2}-\d{2}[Tt ]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$/', $since ) ) {
		return invalid( 'invalid_since', 'since must be an RFC3339 timestamp' );
	}

	$time = strtotime( $since );
	if ( false === $time ) {
		return invalid( 'invalid_since', 'since must be an RFC3339 timestamp' );
	}
	return gmdate( 'Y-m-d H:i:s', $time );
}

function clamp_limit( $limit ): int {
	$value = is_numeric( $limit ) ? (int) $limit : DEFAULT_LIMIT;
	if ( $value < 1 ) {
		return DEFAULT_LIMIT;
	}
	return min( $value, MAX_LIMIT );
}

function content_list( \WP_REST_Request $request ) {
	$types = parse_types( (string) $request->get_param( 'types' ) );
	if ( is_wp_error( $types ) ) {
		return $types;
	}

	$since = parse_since( (string) $request->get_param( 'since' ) );
	if ( is_wp_error( $since ) ) {
		return $since;
	}

	$raw_cursor = (string) $request->get_param( 'cursor' );
	$cursor     = null;
	if ( '' !== $raw_cursor ) {
		$cursor = decode_cursor( $raw_cursor );
		if ( null === $cursor ) {
			return invalid( 'invalid_cursor', 'cursor is not a cursor this endpoint issued' );
		}
	}

	$page = collect( $types, $since, $cursor, clamp_limit( $request->get_param( 'limit' ) ) );

	return new \WP_REST_Response(
		array(
			'items'      => array_values( $page['items'] ),
			'nextCursor' => $page['nextCursor'],
		),
		200
	);
}
