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
	if ( ! isset( $state['i'] ) || ! is_int( $state['i'] ) || $state['i'] < 0 ) {
		return null;
	}
	if ( 'post' === $state['p'] && ( ! isset( $state['m'] ) || ! is_string( $state['m'] ) || '' === $state['m'] ) ) {
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
		$sql   .= ' AND post_modified_gmt > %s';
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

	$name = (string) $publishable->post_name;
	if ( '' === $name ) {
		$name = sanitize_title( (string) $publishable->post_title );
	}
	$publishable->post_name = wp_unique_post_slug(
		$name,
		(int) $publishable->ID,
		'publish',
		(string) $publishable->post_type,
		(int) $publishable->post_parent
	);

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

function query_terms_page( string $since_gmt, ?array $cursor, int $limit ): array {
	global $wpdb;

	$sql  = "SELECT t.term_id AS term_id FROM {$wpdb->term_taxonomy} tt
		INNER JOIN {$wpdb->terms} t ON t.term_id = tt.term_id
		WHERE tt.taxonomy = %s";
	$args = array( 'product_cat' );

	if ( null !== $cursor && isset( $cursor['i'] ) ) {
		$sql   .= ' AND t.term_id > %d';
		$args[] = (int) $cursor['i'];
	}

	$sql   .= ' ORDER BY t.term_id ASC LIMIT %d';
	$args[] = $limit;

	$rows = $wpdb->get_results( $wpdb->prepare( $sql, $args ), ARRAY_A );
	if ( ! is_array( $rows ) ) {
		return array();
	}

	$selected = array();
	foreach ( $rows as $row ) {
		$term_id  = (int) $row['term_id'];
		$modified = term_modified( $term_id );
		if ( '' !== $since_gmt && rfc3339( $since_gmt ) >= $modified ) {
			continue;
		}
		$selected[] = array(
			'term_id'      => $term_id,
			'modified_gmt' => $modified,
		);
	}
	return array( 'rows' => $selected, 'scanned' => count( $rows ), 'last' => empty( $rows ) ? 0 : (int) end( $rows )['term_id'] );
}

function term_item( \WP_Term $term, string $modified ): array {
	$description = (string) $term->description;
	$link        = get_term_link( $term );

	return array(
		'id'          => (int) $term->term_id,
		'type'        => (string) $term->taxonomy,
		'slug'        => (string) $term->slug,
		'path'        => is_string( $link ) ? url_to_path( $link ) : '/',
		'parent'      => (int) $term->parent,
		'status'      => 'publish',
		'modified'    => $modified,
		'contentHash' => content_hash( $description ),
		'title'       => (string) $term->name,
		'h1'          => '',
		'meta'        => read_term_seo( $term ),
		'links'       => array(),
	);
}

function collect( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	$post_types = array_values( array_diff( $types, TERM_TYPES ) );
	$term_types = array_values( array_intersect( $types, TERM_TYPES ) );

	$items = array();
	$phase = null !== $cursor ? (string) $cursor['p'] : ( empty( $post_types ) ? 'term' : 'post' );
	$state = $cursor;

	if ( 'post' === $phase ) {
		$rows = empty( $post_types ) ? array() : query_posts_page( $post_types, $since_gmt, $state, $limit );
		foreach ( $rows as $row ) {
			$post = get_post( (int) $row['ID'] );
			if ( $post instanceof \WP_Post ) {
				$items[] = post_item( $post );
			}
		}
		if ( count( $rows ) === $limit ) {
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
		$state = null;
	}

	$remaining = $limit - count( $items );
	if ( empty( $term_types ) || $remaining < 1 ) {
		return array( 'items' => $items, 'nextCursor' => null );
	}

	$page = query_terms_page( $since_gmt, $state, $remaining );
	foreach ( $page['rows'] as $row ) {
		$term = get_term( (int) $row['term_id'] );
		if ( $term instanceof \WP_Term ) {
			$items[] = term_item( $term, (string) $row['modified_gmt'] );
		}
	}
	if ( $page['scanned'] === $remaining ) {
		return array(
			'items'      => $items,
			'nextCursor' => encode_cursor( array( 'p' => 'term', 'i' => (int) $page['last'] ) ),
		);
	}
	return array( 'items' => $items, 'nextCursor' => null );
}
