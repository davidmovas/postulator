<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function content_hash( string $content ): string {
	return hash( 'sha256', $content );
}

function rfc3339( string $mysql_gmt ): string {
	$time = strtotime( $mysql_gmt . ' UTC' );
	if ( false === $time || $time <= 0 ) {
		return gmdate( 'Y-m-d\TH:i:s\Z', 0 );
	}
	return gmdate( 'Y-m-d\TH:i:s\Z', $time );
}

function now_rfc3339(): string {
	return gmdate( 'Y-m-d\TH:i:s\Z' );
}

function touch_term( int $term_id, int $term_taxonomy_id, string $taxonomy ): void {
	if ( ! in_array( $taxonomy, TERM_TYPES, true ) ) {
		return;
	}
	update_term_meta( $term_id, TERM_MODIFIED_KEY, now_rfc3339() );
}

function term_modified( int $term_id ): string {
	$stored = get_term_meta( $term_id, TERM_MODIFIED_KEY, true );
	if ( is_string( $stored ) && '' !== $stored ) {
		return $stored;
	}

	$now = now_rfc3339();
	update_term_meta( $term_id, TERM_MODIFIED_KEY, $now );
	return $now;
}

function load_dom( string $html ): ?\DOMDocument {
	if ( '' === trim( $html ) || ! class_exists( '\DOMDocument' ) ) {
		return null;
	}

	$document = new \DOMDocument();
	$previous = libxml_use_internal_errors( true );
	$loaded   = $document->loadHTML( '<?xml encoding="UTF-8">' . $html, LIBXML_NOWARNING | LIBXML_NOERROR );
	libxml_clear_errors();
	libxml_use_internal_errors( $previous );

	return $loaded ? $document : null;
}

function rendered_content( \WP_Post $post ): string {
	$previous        = isset( $GLOBALS['post'] ) ? $GLOBALS['post'] : null;
	$GLOBALS['post'] = $post;
	setup_postdata( $post );
	$rendered = (string) apply_filters( 'the_content', $post->post_content );
	wp_reset_postdata();
	$GLOBALS['post'] = $previous;

	return $rendered;
}

function first_h1( string $rendered ): string {
	$document = load_dom( $rendered );
	if ( null !== $document ) {
		$nodes = $document->getElementsByTagName( 'h1' );
		if ( $nodes->length > 0 ) {
			return collapse_text( (string) $nodes->item( 0 )->textContent );
		}
		return '';
	}
	if ( 1 === preg_match( '#<h1\b[^>]*>(.*?)</h1>#is', $rendered, $matches ) ) {
		return collapse_text( html_entity_decode( wp_strip_all_tags( $matches[1] ), ENT_QUOTES | ENT_HTML5, 'UTF-8' ) );
	}
	return '';
}

function extract_links( string $content, string $base ): array {
	$links    = array();
	$document = load_dom( $content );
	if ( null === $document ) {
		return $links;
	}

	foreach ( $document->getElementsByTagName( 'a' ) as $anchor ) {
		$path = internal_href_to_path( (string) $anchor->getAttribute( 'href' ), $base );
		if ( '' === $path ) {
			continue;
		}
		$links[] = array(
			'href'   => $path,
			'anchor' => collapse_text( (string) $anchor->textContent ),
		);
	}
	return $links;
}

function encode_cursor( array $state ): string {
	$json = wp_json_encode( $state );
	if ( ! is_string( $json ) ) {
		return '';
	}
	return rtrim( strtr( base64_encode( $json ), '+/', '-_' ), '=' );
}

function decode_cursor( string $cursor ): ?array {
	$padded  = strtr( $cursor, '-_', '+/' );
	$padded .= str_repeat( '=', ( 4 - ( strlen( $padded ) % 4 ) ) % 4 );

	$raw = base64_decode( $padded, true );
	if ( false === $raw ) {
		return null;
	}
	$state = json_decode( $raw, true );
	if ( ! is_array( $state ) || ! isset( $state['p'] ) || ! in_array( $state['p'], array( 'post', 'term' ), true ) ) {
		return null;
	}
	return $state;
}

function query_posts_page( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	global $wpdb;

	$placeholders = implode( ', ', array_fill( 0, count( $types ), '%s' ) );
	$sql          = "SELECT ID, post_modified_gmt FROM {$wpdb->posts} WHERE post_type IN ( {$placeholders} ) AND post_status NOT IN ( 'auto-draft', 'trash' )";
	$args         = $types;

	if ( '' !== $since_gmt ) {
		$sql   .= ' AND post_modified_gmt >= %s';
		$args[] = $since_gmt;
	}
	if ( null !== $cursor && isset( $cursor['m'], $cursor['i'] ) ) {
		$sql   .= ' AND ( post_modified_gmt > %s OR ( post_modified_gmt = %s AND ID > %d ) )';
		$args[] = (string) $cursor['m'];
		$args[] = (string) $cursor['m'];
		$args[] = (int) $cursor['i'];
	}

	$sql   .= ' ORDER BY post_modified_gmt ASC, ID ASC LIMIT %d';
	$args[] = $limit;

	$rows = $wpdb->get_results( $wpdb->prepare( $sql, $args ), ARRAY_A );
	return is_array( $rows ) ? $rows : array();
}

function permalink_path( \WP_Post $post ): string {
	if ( ! in_array( $post->post_status, array( 'draft', 'pending', 'auto-draft', 'future' ), true ) ) {
		return url_to_path( (string) get_permalink( $post ) );
	}

	$publishable              = clone $post;
	$publishable->post_status = 'publish';
	if ( '' === (string) $publishable->post_name ) {
		$publishable->post_name = sanitize_title( (string) $publishable->post_title, (int) $publishable->ID );
	}
	return url_to_path( (string) get_permalink( $publishable ) );
}

function post_item( \WP_Post $post ): array {
	$path    = permalink_path( $post );
	$content = (string) $post->post_content;

	return array(
		'id'          => (int) $post->ID,
		'type'        => (string) $post->post_type,
		'slug'        => (string) $post->post_name,
		'path'        => $path,
		'parent'      => (int) $post->post_parent,
		'status'      => (string) $post->post_status,
		'modified'    => rfc3339( (string) $post->post_modified_gmt ),
		'contentHash' => content_hash( $content ),
		'title'       => (string) $post->post_title,
		'h1'          => first_h1( rendered_content( $post ) ),
		'meta'        => read_post_seo( (int) $post->ID ),
		'links'       => extract_links( $content, parent_path( $path ) ),
	);
}

function collect( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	$items = array();

	$rows = query_posts_page( $types, $since_gmt, $cursor, $limit );
	foreach ( $rows as $row ) {
		$post = get_post( (int) $row['ID'] );
		if ( $post instanceof \WP_Post ) {
			$items[] = post_item( $post );
		}
	}
	if ( count( $rows ) < $limit ) {
		return array( 'items' => $items, 'nextCursor' => null );
	}

	$last = end( $rows );
	return array(
		'items'      => $items,
		'nextCursor' => encode_cursor(
			array(
				'p' => 'post',
				'm' => (string) $last['post_modified_gmt'],
				'i' => (int) $last['ID'],
			)
		),
	);
}
