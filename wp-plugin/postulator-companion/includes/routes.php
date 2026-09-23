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

	register_rest_route(
		NAMESPACE_PATH,
		'/seo-meta/(?P<id>\d+)',
		array(
			array(
				'methods'             => \WP_REST_Server::READABLE,
				'callback'            => __NAMESPACE__ . '\\seo_meta_get',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
			array(
				'methods'             => \WP_REST_Server::EDITABLE,
				'callback'            => __NAMESPACE__ . '\\seo_meta_update',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
		)
	);

	register_rest_route(
		NAMESPACE_PATH,
		'/content/(?P<id>\d+)/raw',
		array(
			array(
				'methods'             => \WP_REST_Server::READABLE,
				'callback'            => __NAMESPACE__ . '\\raw_get',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
			array(
				'methods'             => \WP_REST_Server::EDITABLE,
				'callback'            => __NAMESPACE__ . '\\raw_update',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
		)
	);

	register_rest_route(
		NAMESPACE_PATH,
		'/content/(?P<id>\d+)/preview',
		array(
			'methods'             => \WP_REST_Server::CREATABLE,
			'callback'            => __NAMESPACE__ . '\\preview_create',
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

function text_param( \WP_REST_Request $request, string $name ) {
	$value = $request->get_param( $name );
	if ( null === $value ) {
		return '';
	}
	if ( ! is_string( $value ) ) {
		return invalid( 'invalid_param', $name . ' must be a string' );
	}
	return trim( $value );
}

function content_list( \WP_REST_Request $request ) {
	$raw_types = text_param( $request, 'types' );
	if ( is_wp_error( $raw_types ) ) {
		return $raw_types;
	}
	$types = parse_types( $raw_types );
	if ( is_wp_error( $types ) ) {
		return $types;
	}

	$raw_since = text_param( $request, 'since' );
	if ( is_wp_error( $raw_since ) ) {
		return $raw_since;
	}
	$since = parse_since( $raw_since );
	if ( is_wp_error( $since ) ) {
		return $since;
	}

	$raw_cursor = text_param( $request, 'cursor' );
	if ( is_wp_error( $raw_cursor ) ) {
		return $raw_cursor;
	}
	$cursor = null;
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

function editable_post( \WP_REST_Request $request ) {
	$post = get_post( (int) $request['id'] );
	if ( ! $post instanceof \WP_Post || in_array( $post->post_status, array( 'auto-draft', 'trash' ), true ) ) {
		return not_found( 'no post with that id' );
	}
	if ( ! current_user_can( 'edit_post', $post->ID ) ) {
		return forbidden( 'editing this post is not allowed' );
	}
	return $post;
}

function seo_meta_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	return new \WP_REST_Response(
		array_merge(
			read_post_seo_all( (int) $post->ID ),
			array( 'seoPlugin' => detect_plugin() )
		),
		200
	);
}

function seo_meta_update( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$body = $request->get_json_params();
	if ( ! is_array( $body ) ) {
		return invalid( 'invalid_body', 'a JSON object body is required' );
	}

	$fields = array();
	foreach ( array_keys( META_KEYS['none'] ) as $field ) {
		if ( ! array_key_exists( $field, $body ) ) {
			continue;
		}
		if ( ! is_string( $body[ $field ] ) ) {
			return invalid( 'invalid_param', $field . ' must be a string' );
		}
		$fields[ $field ] = $body[ $field ];
	}

	return new \WP_REST_Response(
		array(
			'applied'   => write_post_seo( (int) $post->ID, $fields ),
			'seoPlugin' => detect_plugin(),
		),
		200
	);
}

function raw_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$content = (string) $post->post_content;
	return new \WP_REST_Response(
		array(
			'id'          => (int) $post->ID,
			'type'        => (string) $post->post_type,
			'content'     => $content,
			'contentHash' => content_hash( $content ),
		),
		200
	);
}

function raw_update( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$body = $request->get_json_params();
	if ( ! is_array( $body ) || ! array_key_exists( 'content', $body ) || ! is_string( $body['content'] ) ) {
		return invalid( 'invalid_body', 'content is required and must be a string' );
	}

	$current  = (string) $post->post_content;
	$expected = isset( $body['expectedHash'] ) && is_string( $body['expectedHash'] ) ? $body['expectedHash'] : '';
	if ( '' !== $expected && ! hash_equals( content_hash( $current ), $expected ) ) {
		return new \WP_REST_Response(
			array(
				'code'        => 'hash_mismatch',
				'message'     => 'the stored content does not match expectedHash',
				'currentHash' => content_hash( $current ),
			),
			409
		);
	}

	$updated = wp_update_post(
		array(
			'ID'           => (int) $post->ID,
			'post_content' => wp_slash( (string) $body['content'] ),
		),
		true
	);
	if ( is_wp_error( $updated ) ) {
		return failed( $updated->get_error_message() );
	}

	$stored = (string) get_post_field( 'post_content', (int) $post->ID, 'raw' );
	return new \WP_REST_Response( array( 'contentHash' => content_hash( $stored ) ), 200 );
}

function preview_create( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$issued = issue_preview( $post );
	if ( is_wp_error( $issued ) ) {
		return $issued;
	}
	return new \WP_REST_Response( $issued, 200 );
}
