<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function boot_head(): void {
	if ( 'none' !== detect_plugin() ) {
		return;
	}
	add_filter( 'pre_get_document_title', __NAMESPACE__ . '\\filter_document_title' );
	add_filter( 'get_canonical_url', __NAMESPACE__ . '\\filter_canonical_url', 10, 2 );
	add_action( 'wp_head', __NAMESPACE__ . '\\print_head_tags', 2 );
}

function head_object_id(): int {
	return is_singular() ? (int) get_queried_object_id() : 0;
}

function filter_document_title( $title ) {
	$id = head_object_id();
	if ( 0 === $id ) {
		return $title;
	}
	$own = (string) get_post_meta( $id, META_KEYS['none']['title'], true );
	return '' === $own ? $title : $own;
}

function filter_canonical_url( $canonical, $post ) {
	if ( ! $post instanceof \WP_Post ) {
		return $canonical;
	}
	$own = (string) get_post_meta( (int) $post->ID, META_KEYS['none']['canonical'], true );
	return '' === $own ? $canonical : $own;
}

function print_head_tags(): void {
	static $printed = false;

	if ( $printed ) {
		return;
	}
	$id = head_object_id();
	if ( 0 === $id ) {
		return;
	}
	$printed = true;

	$map  = META_KEYS['none'];
	$tags = array(
		'description'   => array( 'name', 'description' ),
		'ogTitle'       => array( 'property', 'og:title' ),
		'ogDescription' => array( 'property', 'og:description' ),
	);

	foreach ( $tags as $field => $tag ) {
		$value = (string) get_post_meta( $id, $map[ $field ], true );
		if ( '' === $value ) {
			continue;
		}
		printf(
			'<meta %s="%s" content="%s" />' . "\n",
			esc_attr( $tag[0] ),
			esc_attr( $tag[1] ),
			esc_attr( $value )
		);
	}
}
