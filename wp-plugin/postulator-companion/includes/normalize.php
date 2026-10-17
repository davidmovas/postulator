<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function site_authority(): string {
	if ( 1 !== preg_match( '#^[A-Za-z][A-Za-z0-9+.\-]*://([^/?\#]*)#', (string) home_url(), $matches ) ) {
		return '';
	}
	return host_of_authority( $matches[1] );
}

function host_of_authority( string $authority ): string {
	$at = strrpos( $authority, '@' );
	if ( false !== $at ) {
		$authority = substr( $authority, $at + 1 );
	}
	return strtolower( $authority );
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

function href_base( string $url ): string {
	$path = wp_parse_url( $url, PHP_URL_PATH );
	if ( ! is_string( $path ) || '' === $path || '/' !== $path[0] ) {
		return '/';
	}
	$escaped = escaped_path( substr( $path, 0, (int) strrpos( $path, '/' ) + 1 ) );
	return null === $escaped ? '/' : $escaped;
}

function escaped_path( string $raw ): ?string {
	if ( 1 === preg_match( '/%(?![0-9A-Fa-f]{2})/', $raw ) ) {
		return null;
	}
	if ( 1 === preg_match( "#^[A-Za-z0-9\\-._~!$&'()*+,;=:@\\[\\]/%]*\\z#", $raw ) ) {
		return $raw;
	}
	return (string) preg_replace_callback(
		"#[^A-Za-z0-9\\-._~$&+,/:;=@]#",
		function ( array $matches ): string {
			return '%' . strtoupper( bin2hex( $matches[0] ) );
		},
		rawurldecode( $raw )
	);
}

function href_path( string $escaped ): string {
	$segments = array();
	foreach ( explode( '/', $escaped ) as $segment ) {
		if ( '' === $segment ) {
			continue;
		}
		if ( '.' === $segment || '..' === $segment ) {
			return '';
		}
		$segments[] = strtolower( $segment );
	}
	if ( empty( $segments ) ) {
		return '/';
	}
	return '/' . implode( '/', $segments ) . '/';
}

function internal_href_to_path( string $href, string $base ): string {
	$href = trim( $href );
	if ( '' === $href || '#' === $href[0] || '?' === $href[0] ) {
		return '';
	}
	if ( 1 === preg_match( '/[\x00-\x1f\x7f]/', $href ) ) {
		return '';
	}

	$rest       = substr( $href, 0, strcspn( $href, '#' ) );
	$has_scheme = false;
	if ( 1 === preg_match( '#^([A-Za-z][A-Za-z0-9+.\-]*):(.*)\z#s', $rest, $matches ) ) {
		if ( ! in_array( strtolower( $matches[1] ), array( 'http', 'https' ), true ) ) {
			return '';
		}
		$has_scheme = true;
		$rest       = $matches[2];
	}
	$rest = substr( $rest, 0, strcspn( $rest, '?' ) );

	$has_host = false;
	if ( 0 === strpos( $rest, '//' ) && ( $has_scheme || 0 !== strpos( $rest, '///' ) ) ) {
		$length = strcspn( $rest, '/', 2 );
		if ( host_of_authority( substr( $rest, 2, $length ) ) !== site_authority() ) {
			return '';
		}
		$has_host = true;
		$rest     = (string) substr( $rest, 2 + $length );
	} elseif ( $has_scheme && ( '' === $rest || '/' !== $rest[0] ) ) {
		return '';
	}

	if ( '' === $rest ) {
		return $has_host ? '/' : '';
	}

	$escaped = escaped_path( $rest );
	if ( null === $escaped ) {
		return '';
	}
	if ( '/' !== $escaped[0] ) {
		if ( $has_host ) {
			return '';
		}
		$escaped = $base . $escaped;
	}
	return href_path( $escaped );
}

function collapse_text( string $text ): string {
	return trim( (string) preg_replace( '/\s+/u', ' ', $text ) );
}
