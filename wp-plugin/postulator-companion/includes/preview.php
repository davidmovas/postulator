<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function preview_query_vars( array $vars ): array {
	$vars[] = PREVIEW_QUERY_VAR;
	return $vars;
}

function preview_query( \WP_Query $query ): void {
	if ( is_admin() || ! $query->is_main_query() ) {
		return;
	}
	$token = $query->get( PREVIEW_QUERY_VAR );
	if ( ! is_string( $token ) || '' === $token ) {
		return;
	}
	$query->set( 'cache_results', false );
	add_filter( 'posts_results', __NAMESPACE__ . '\\preview_posts', 10, 2 );
}

function preview_posts( $posts, $query ) {
	if ( ! $query instanceof \WP_Query || ! $query->is_main_query() ) {
		return $posts;
	}
	remove_filter( 'posts_results', __NAMESPACE__ . '\\preview_posts', 10 );

	if ( ! is_array( $posts ) || 1 !== count( $posts ) || ! $query->is_singular() ) {
		return $posts;
	}
	$post = $posts[0];
	if ( ! $post instanceof \WP_Post || ! in_array( $post->post_status, PREVIEW_STATUSES, true ) ) {
		return $posts;
	}
	if ( ! preview_valid( (int) $post->ID, (string) $query->get( PREVIEW_QUERY_VAR ) ) ) {
		return $posts;
	}

	$post->post_status = 'publish';
	preview_headers();
	return $posts;
}

function preview_valid( int $post_id, string $token ): bool {
	if ( 1 !== preg_match( '/^[0-9a-f]{32}$/', $token ) ) {
		return false;
	}
	$stored  = (string) get_post_meta( $post_id, PREVIEW_HASH_KEY, true );
	$expires = (int) get_post_meta( $post_id, PREVIEW_EXPIRES_KEY, true );
	if ( '' === $stored || $expires <= time() ) {
		return false;
	}
	return hash_equals( $stored, hash( 'sha256', $token ) );
}

function preview_headers(): void {
	if ( ! headers_sent() ) {
		nocache_headers();
		header( 'X-Robots-Tag: noindex, nofollow', true );
	}
	add_filter( 'wp_robots', 'wp_robots_no_robots' );
	add_filter( 'redirect_canonical', '__return_false' );
	add_filter( 'comments_open', '__return_false' );
	add_filter( 'pings_open', '__return_false' );
	if ( ! defined( 'DONOTCACHEPAGE' ) ) {
		define( 'DONOTCACHEPAGE', true );
	}
}

function issue_preview( \WP_Post $post ) {
	try {
		$token = bin2hex( random_bytes( 16 ) );
	} catch ( \Exception $cause ) {
		return failed( 'the site has no source of randomness' );
	}
	$expires = time() + PREVIEW_TTL;

	update_post_meta( (int) $post->ID, PREVIEW_HASH_KEY, wp_slash( hash( 'sha256', $token ) ) );
	update_post_meta( (int) $post->ID, PREVIEW_EXPIRES_KEY, $expires );

	return array(
		'url'       => preview_url( $post, $token ),
		'expiresAt' => gmdate( 'Y-m-d\TH:i:s\Z', $expires ),
	);
}

function preview_url( \WP_Post $post, string $token ): string {
	return add_query_arg(
		array(
			'preview'         => 'true',
			PREVIEW_QUERY_VAR => $token,
		),
		get_permalink( $post )
	);
}
