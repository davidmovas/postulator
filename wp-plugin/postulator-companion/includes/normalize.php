<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function site_host(): string {
	$host = wp_parse_url( home_url(), PHP_URL_HOST );
	return is_string( $host ) ? strtolower( $host ) : '';
}

function normalize_path( string $path ): string {
	$path = (string) preg_replace( '#/+#', '/', $path );
	if ( '' === $path ) {
		return '/';
	}
	if ( '/' !== $path[0] ) {
		$path = '/' . $path;
	}
	if ( '/' !== substr( $path, -1 ) ) {
		$path .= '/';
	}
	return strtolower( $path );
}

function url_to_path( string $url ): string {
	$path = wp_parse_url( $url, PHP_URL_PATH );
	return normalize_path( is_string( $path ) ? $path : '/' );
}

function parent_path( string $path ): string {
	$trimmed = rtrim( $path, '/' );
	$cut     = strrpos( $trimmed, '/' );
	if ( false === $cut ) {
		return '/';
	}
	return normalize_path( substr( $trimmed, 0, $cut + 1 ) );
}

function internal_href_to_path( string $href, string $base ): string {
	$href = trim( $href );
	if ( '' === $href || '#' === $href[0] ) {
		return '';
	}

	$parts = wp_parse_url( $href );
	if ( ! is_array( $parts ) ) {
		return '';
	}
	if ( isset( $parts['scheme'] ) && ! in_array( strtolower( $parts['scheme'] ), array( 'http', 'https' ), true ) ) {
		return '';
	}
	if ( isset( $parts['host'] ) && strtolower( $parts['host'] ) !== site_host() ) {
		return '';
	}

	$path = isset( $parts['path'] ) ? $parts['path'] : '';
	if ( '' === $path ) {
		return isset( $parts['host'] ) ? '/' : '';
	}
	if ( '/' !== $path[0] ) {
		if ( isset( $parts['host'] ) ) {
			return '';
		}
		$path = $base . $path;
	}
	return normalize_path( $path );
}

function collapse_text( string $text ): string {
	return trim( (string) preg_replace( '/\s+/u', ' ', $text ) );
}
