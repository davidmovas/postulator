<?php

add_action( 'init', 'shop_register_entity_taxonomy' );
add_action( 'template_redirect', 'shop_legacy_category_redirect', 9 );
add_filter( 'the_content', 'shop_hub_content' );
add_action( 'wp_head', 'shop_hub_schema' );

function shop_register_entity_taxonomy() {
	register_taxonomy(
		'entity',
		array( 'product' ),
		array(
			'labels'            => array(
				'name'          => 'Entities',
				'singular_name' => 'Entity',
			),
			'public'            => true,
			'hierarchical'      => false,
			'show_ui'           => true,
			'show_admin_column' => true,
			'rewrite'           => array( 'slug' => 'entity' ),
		)
	);
}

function shop_hub_crumbs() {
	return array(
		'peptides'         => array(
			'crumbs' => array(),
			'schema' => true,
		),
		'healing-peptides' => array(
			'crumbs' => array(
				array( 'Peptides', '/peptides/' ),
			),
			'schema' => true,
		),
		'bpc-157'          => array(
			'crumbs' => array(
				array( 'Peptides', '/product-category/peptides/' ),
				array( 'Healing Peptides', '/product-category/peptides/healing-peptides/' ),
			),
			'schema' => true,
		),
		'tb-500'           => array(
			'crumbs' => array(
				array( 'Peptides', '' ),
				array( 'Healing Peptides', '/category/healing-peptides/' ),
			),
			'schema' => true,
		),
		'reconstitution'   => array(
			'crumbs' => array(
				array( 'Lab Guides', '' ),
			),
			'schema' => false,
		),
	);
}

function shop_legacy_category_redirect() {
	if ( ! is_404() ) {
		return;
	}
	$path = (string) wp_parse_url( isset( $_SERVER['REQUEST_URI'] ) ? wp_unslash( $_SERVER['REQUEST_URI'] ) : '', PHP_URL_PATH );
	if ( ! preg_match( '#^/category/([a-z0-9-]+)/?$#', $path, $matches ) ) {
		return;
	}
	$hub = get_page_by_path( $matches[1] );
	if ( ! $hub instanceof WP_Post || 'publish' !== $hub->post_status ) {
		return;
	}
	wp_safe_redirect( get_permalink( $hub ), 301 );
	exit;
}

function shop_hub_term( $post ) {
	if ( ! $post instanceof WP_Post || 'page' !== $post->post_type ) {
		return null;
	}
	foreach ( array( 'product_cat', 'entity' ) as $taxonomy ) {
		$term = get_term_by( 'slug', $post->post_name, $taxonomy );
		if ( $term instanceof WP_Term ) {
			return $term;
		}
	}
	return null;
}

function shop_hub_entry( $post ) {
	if ( ! $post instanceof WP_Post || 'page' !== $post->post_type ) {
		return null;
	}
	$table = shop_hub_crumbs();
	return isset( $table[ $post->post_name ] ) ? $table[ $post->post_name ] : null;
}

function shop_hub_trail( WP_Post $post, array $entry ) {
	$crumbs = array(
		array(
			'name' => 'Home',
			'url'  => home_url( '/' ),
		),
	);
	foreach ( $entry['crumbs'] as $crumb ) {
		$crumbs[] = array(
			'name' => $crumb[0],
			'url'  => '' === $crumb[1] ? '' : home_url( $crumb[1] ),
		);
	}
	$crumbs[] = array(
		'name' => get_the_title( $post ),
		'url'  => get_permalink( $post ),
	);
	return $crumbs;
}

function shop_hub_products( WP_Term $term ) {
	return get_posts(
		array(
			'post_type'   => 'product',
			'post_status' => 'publish',
			'numberposts' => 12,
			'orderby'     => 'title',
			'order'       => 'ASC',
			'tax_query'   => array(
				array(
					'taxonomy'         => $term->taxonomy,
					'field'            => 'term_id',
					'terms'            => $term->term_id,
					'include_children' => true,
				),
			),
		)
	);
}

function shop_breadcrumb_nav( array $crumbs ) {
	$items = '';
	$last  = count( $crumbs ) - 1;
	foreach ( $crumbs as $index => $crumb ) {
		if ( $index === $last ) {
			$items .= '<li><span aria-current="page">' . esc_html( $crumb['name'] ) . '</span></li>';
			continue;
		}
		if ( '' === $crumb['url'] ) {
			$items .= '<li><span>' . esc_html( $crumb['name'] ) . '</span></li>';
			continue;
		}
		$items .= '<li><a href="' . esc_url( $crumb['url'] ) . '">' . esc_html( $crumb['name'] ) . '</a></li>';
	}
	return '<nav class="hub-breadcrumbs" aria-label="Breadcrumb"><ol>' . $items . '</ol></nav>';
}

function shop_product_section( WP_Term $term ) {
	$cards = '';
	foreach ( shop_hub_products( $term ) as $product ) {
		$cards .= '<article class="entity-auto-product"><a href="' . esc_url( get_permalink( $product ) ) . '">'
			. esc_html( get_the_title( $product ) ) . '</a></article>';
	}
	return '<section class="entity-auto-products"><h2>' . esc_html( $term->name ) . ' products</h2>'
		. '<div class="entity-auto-products-grid">' . $cards . '</div></section>';
}

function shop_hub_content( $content ) {
	if ( ! ( is_singular() && in_the_loop() && is_main_query() ) ) {
		return $content;
	}
	$post  = get_post();
	$entry = shop_hub_entry( $post );
	$term  = shop_hub_term( $post );
	if ( null !== $entry ) {
		$content = shop_breadcrumb_nav( shop_hub_trail( $post, $entry ) ) . $content;
	}
	if ( null !== $term ) {
		$content .= shop_product_section( $term );
	}
	return $content;
}

function shop_trail_schema( WP_Post $post, array $entry ) {
	$url      = get_permalink( $post );
	$elements = array();
	foreach ( shop_hub_trail( $post, $entry ) as $index => $crumb ) {
		$element = array(
			'@type'    => 'ListItem',
			'position' => $index + 1,
			'name'     => $crumb['name'],
		);
		if ( '' !== $crumb['url'] ) {
			$element['item'] = $crumb['url'];
		}
		$elements[] = $element;
	}
	return array(
		'@context'        => 'https://schema.org',
		'@type'           => 'BreadcrumbList',
		'@id'             => $url . '#trail',
		'itemListElement' => $elements,
	);
}

function shop_hub_schema() {
	if ( ! is_page() ) {
		return;
	}
	$post  = get_queried_object();
	$entry = shop_hub_entry( $post );
	$term  = shop_hub_term( $post );
	$graph = array();
	if ( null !== $entry && $entry['schema'] ) {
		$graph[] = shop_trail_schema( $post, $entry );
	}
	if ( null !== $term ) {
		$url   = get_permalink( $post );
		$items = array();
		foreach ( shop_hub_products( $term ) as $index => $product ) {
			$items[] = array(
				'@type'    => 'ListItem',
				'position' => $index + 1,
				'name'     => get_the_title( $product ),
				'url'      => get_permalink( $product ),
			);
		}
		$graph[] = array(
			'@context'   => 'https://schema.org',
			'@type'      => 'CollectionPage',
			'@id'        => $url . '#collection',
			'url'        => $url,
			'name'       => get_the_title( $post ),
			'breadcrumb' => array( '@id' => $url . '#trail' ),
			'mainEntity' => array( '@id' => $url . '#listing' ),
		);
		$graph[] = array(
			'@context'        => 'https://schema.org',
			'@type'           => 'ItemList',
			'@id'             => $url . '#listing',
			'numberOfItems'   => count( $items ),
			'itemListElement' => $items,
		);
	}
	foreach ( $graph as $node ) {
		echo '<script type="application/ld+json">' . wp_json_encode( $node ) . '</script>' . "\n";
	}
}
