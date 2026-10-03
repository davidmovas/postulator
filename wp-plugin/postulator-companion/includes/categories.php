<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function register_page_categories(): void {
	register_taxonomy_for_object_type( 'category', 'page' );
}

function category_archive_query( \WP_Query $query ): void {
	if ( is_admin() || ! $query->is_main_query() || ! $query->is_category() ) {
		return;
	}

	$current = $query->get( 'post_type' );
	if ( 'any' === $current || ( is_array( $current ) && in_array( 'any', $current, true ) ) ) {
		return;
	}

	$types = array();
	foreach ( (array) $current as $type ) {
		if ( is_string( $type ) && '' !== $type && ! in_array( $type, $types, true ) ) {
			$types[] = $type;
		}
	}
	if ( array() === $types ) {
		$types[] = 'post';
	}
	foreach ( ARCHIVE_TYPES as $type ) {
		if ( ! in_array( $type, $types, true ) ) {
			$types[] = $type;
		}
	}

	$query->set( 'post_type', $types );
}
