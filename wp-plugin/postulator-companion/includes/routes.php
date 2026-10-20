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
		'/content/(?P<id>\d+)/rendered',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\rendered_get',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);

	register_rest_route(
		NAMESPACE_PATH,
		'/meta/(?P<id>\d+)',
		array(
			array(
				'methods'             => \WP_REST_Server::READABLE,
				'callback'            => __NAMESPACE__ . '\\meta_get',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
			array(
				'methods'             => \WP_REST_Server::EDITABLE,
				'callback'            => __NAMESPACE__ . '\\meta_put',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
		)
	);

	register_rest_route(
		NAMESPACE_PATH,
		'/settings',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\settings_get',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);

	register_rest_route(
		NAMESPACE_PATH,
		'/terms/(?P<id>\d+)',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\terms_get',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
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
			'version'        => VERSION,
			'capabilities'   => CAPABILITIES,
			'postTypes'      => content_post_types(),
			'taxonomies'     => content_taxonomies(),
			'seoPlugin'      => detect_plugin(),
			'wpVersion'      => (string) get_bloginfo( 'version' ),
			'site'           => (string) home_url(),
			'unfilteredHtml' => current_user_can( 'unfiltered_html' ),
		),
		200
	);
}

function parse_types( string $types ) {
	if ( '' === $types ) {
		return TYPES;
	}

	$requested = array_values( array_filter( array_map( 'trim', explode( ',', $types ) ), 'strlen' ) );
	$listable  = array_merge( TYPES, content_post_types(), content_taxonomies() );
	foreach ( $requested as $type ) {
		if ( ! in_array( $type, $listable, true ) ) {
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

function content_post_types(): array {
	$types = get_post_types( array( 'public' => true ), 'names' );
	unset( $types['attachment'] );
	$types = array_values( $types );
	sort( $types );
	return $types;
}

function content_taxonomies(): array {
	$posts      = content_post_types();
	$taxonomies = array();
	foreach ( get_taxonomies( array( 'public' => true ), 'names' ) as $taxonomy ) {
		if ( ! in_array( $taxonomy, $posts, true ) ) {
			$taxonomies[] = (string) $taxonomy;
		}
	}
	sort( $taxonomies );
	return $taxonomies;
}

function term_families(): array {
	return array_values( array_unique( array_merge( TERM_TYPES, content_taxonomies() ) ) );
}

function editable_post( \WP_REST_Request $request ) {
	$post = get_post( (int) $request['id'] );
	if ( ! $post instanceof \WP_Post || in_array( $post->post_status, array( 'auto-draft', 'trash' ), true ) ) {
		return not_found( 'no post with that id' );
	}
	if ( ! in_array( $post->post_type, content_post_types(), true ) ) {
		return not_found( 'no content post with that id' );
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

	$expected = array();
	if ( array_key_exists( 'expected', $body ) ) {
		if ( ! is_array( $body['expected'] ) ) {
			return invalid( 'invalid_param', 'expected must be an object of hashes' );
		}
		$expected = $body['expected'];
	}
	$moved = seo_moved( (int) $post->ID, $fields, $expected );
	if ( ! empty( $moved ) ) {
		return new \WP_REST_Response(
			array(
				'code'    => 'hash_mismatch',
				'message' => 'an SEO field changed since it was read',
				'moved'   => (object) $moved,
			),
			409
		);
	}

	$written = write_post_seo( (int) $post->ID, $fields );
	return new \WP_REST_Response(
		array(
			'applied'   => $written['applied'],
			'hashes'    => (object) $written['hashes'],
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
			'saveStable'  => save_stable( $content ),
		),
		200
	);
}

function save_stable( string $content ): bool {
	$filtered = apply_filters( 'pre_post_content', wp_slash( $content ) );
	$filtered = apply_filters( 'content_save_pre', $filtered );
	return is_string( $filtered ) && wp_unslash( $filtered ) === $content;
}

function rendered_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$content = filtered_content( $post );
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

function meta_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$raw = text_param( $request, 'keys' );
	if ( is_wp_error( $raw ) ) {
		return $raw;
	}
	$keys = array_values( array_unique( array_filter( array_map( 'trim', explode( ',', $raw ) ), 'strlen' ) ) );
	if ( empty( $keys ) ) {
		return invalid( 'invalid_param', 'keys must name at least one meta key' );
	}
	if ( count( $keys ) > MAX_META_KEYS ) {
		return invalid( 'too_many_keys', 'keys names more than ' . MAX_META_KEYS . ' meta keys' );
	}

	$denied = array();
	foreach ( $keys as $key ) {
		if ( is_protected_meta( $key, 'post' ) || meta_key_denied( $key ) ) {
			$denied[] = $key;
		}
	}
	if ( ! empty( $denied ) ) {
		return denied_meta( $denied, 'read' );
	}

	$values = array();
	foreach ( $keys as $key ) {
		$values[ $key ] = metadata_exists( 'post', (int) $post->ID, $key ) ? get_post_meta( (int) $post->ID, $key, true ) : null;
	}
	return new \WP_REST_Response(
		array(
			'id'   => (int) $post->ID,
			'meta' => (object) $values,
		),
		200
	);
}

function meta_put( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$body = $request->get_json_params();
	if ( ! is_array( $body ) || ! isset( $body['meta'], $body['expected'] ) || ! is_array( $body['meta'] ) || ! is_array( $body['expected'] ) || empty( $body['meta'] ) ) {
		return invalid( 'invalid_body', 'meta and expected are required objects' );
	}
	$values   = array();
	$expected = $body['expected'];
	foreach ( $body['meta'] as $key => $value ) {
		$values[ (string) $key ] = $value;
	}
	if ( count( $values ) > MAX_META_KEYS ) {
		return invalid( 'too_many_keys', 'meta names more than ' . MAX_META_KEYS . ' meta keys' );
	}

	$post_id = (int) $post->ID;
	$denied  = array();
	foreach ( $values as $key => $value ) {
		if ( ! array_key_exists( $key, $expected ) ) {
			return invalid( 'missing_expected', 'every key written names the hash it was read with: ' . $key );
		}
		$text = null === $value || is_string( $value );
		if ( ! $text || meta_key_denied( $key ) || is_protected_meta( $key, 'post' ) || ! current_user_can( 'edit_post_meta', $post_id, $key ) ) {
			$denied[] = $key;
		}
	}
	if ( ! empty( $denied ) ) {
		return denied_meta( $denied, 'written' );
	}

	$moved = array();
	foreach ( array_keys( $values ) as $key ) {
		$state = meta_state( $post_id, $key );
		if ( ! $state['scalar'] ) {
			$denied[] = $key;
			continue;
		}
		if ( meta_moved( $expected[ $key ], $state['hash'] ) ) {
			$moved[ $key ] = $state['hash'];
		}
	}
	if ( ! empty( $denied ) ) {
		return denied_meta( $denied, 'written over a value that is not one plain text' );
	}
	if ( ! empty( $moved ) ) {
		return new \WP_REST_Response(
			array(
				'code'    => 'hash_mismatch',
				'message' => 'a meta value changed since it was read',
				'moved'   => (object) $moved,
			),
			409
		);
	}

	$stored = array();
	foreach ( $values as $key => $value ) {
		if ( null === $value ) {
			delete_post_meta( $post_id, $key );
		} else {
			update_post_meta( $post_id, $key, wp_slash( $value ) );
		}
		$state          = meta_state( $post_id, $key );
		$stored[ $key ] = array(
			'value' => $state['value'],
			'hash'  => $state['hash'],
		);
	}
	return new \WP_REST_Response(
		array(
			'id'   => $post_id,
			'meta' => (object) $stored,
		),
		200
	);
}

function settings_get(): \WP_REST_Response {
	$show_on_front = 'page' === get_option( 'show_on_front' ) ? 'page' : 'posts';
	$pages         = array(
		'front'    => 'page' === $show_on_front ? (int) get_option( 'page_on_front', 0 ) : 0,
		'posts'    => 'page' === $show_on_front ? (int) get_option( 'page_for_posts', 0 ) : 0,
		'privacy'  => (int) get_option( 'wp_page_for_privacy_policy', 0 ),
		'shop'     => 0,
		'cart'     => 0,
		'checkout' => 0,
		'account'  => 0,
		'terms'    => 0,
	);
	if ( function_exists( 'wc_get_page_id' ) ) {
		$store = array(
			'shop'     => 'shop',
			'cart'     => 'cart',
			'checkout' => 'checkout',
			'account'  => 'myaccount',
			'terms'    => 'terms',
		);
		foreach ( $store as $field => $name ) {
			$pages[ $field ] = max( 0, (int) wc_get_page_id( $name ) );
		}
	}

	return new \WP_REST_Response(
		array(
			'showOnFront' => $show_on_front,
			'pages'       => $pages,
		),
		200
	);
}

function terms_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	return new \WP_REST_Response(
		array(
			'id'    => (int) $post->ID,
			'terms' => post_terms( $post ),
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
	return new \WP_REST_Response(
		array(
			'content'     => $stored,
			'contentHash' => content_hash( $stored ),
		),
		200
	);
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
