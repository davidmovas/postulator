<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const DENIED_META_KEYS     = array( '_price', '_regular_price', '_sale_price', '_sku', '_stock_status', '_stock', '_manage_stock', '_product_attributes', 'total_sales' );
const DENIED_META_PREFIXES = array( '_', 'attribute_', 'rank_math_' );

function meta_key_denied( string $key ): bool {
	if ( 1 !== preg_match( '/\A[A-Za-z0-9_-]{1,255}\z/', $key ) ) {
		return true;
	}
	$folded = strtolower( $key );
	if ( in_array( $folded, DENIED_META_KEYS, true ) ) {
		return true;
	}
	foreach ( DENIED_META_PREFIXES as $prefix ) {
		if ( 0 === strpos( $folded, $prefix ) ) {
			return true;
		}
	}
	return false;
}

function meta_state( int $post_id, string $key ): array {
	if ( ! metadata_exists( 'post', $post_id, $key ) ) {
		return array(
			'scalar' => true,
			'value'  => null,
			'hash'   => null,
		);
	}
	$rows = get_post_meta( $post_id, $key, false );
	if ( ! is_array( $rows ) || 1 !== count( $rows ) || ! is_scalar( $rows[0] ) ) {
		return array(
			'scalar' => false,
			'value'  => null,
			'hash'   => null,
		);
	}
	$value = (string) $rows[0];
	return array(
		'scalar' => true,
		'value'  => $value,
		'hash'   => content_hash( $value ),
	);
}

function meta_moved( $expected, $current ): bool {
	if ( null === $expected || null === $current ) {
		return $expected !== $current;
	}
	return ! is_string( $expected ) || ! hash_equals( $current, $expected );
}

function denied_meta( array $keys, string $verb ): \WP_Error {
	sort( $keys );
	return new \WP_Error(
		'meta_key_denied',
		'a protected meta key is never ' . $verb . ': ' . implode( ', ', $keys ),
		array(
			'status' => 400,
			'keys'   => $keys,
		)
	);
}
