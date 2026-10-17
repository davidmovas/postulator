<?php

add_action( 'init', 'shop_register_entity_taxonomy' );
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

function shop_entity_category( WP_Term $entity ) {
	$products = get_posts(
		array(
			'post_type'   => 'product',
			'post_status' => 'publish',
			'numberposts' => 1,
			'orderby'     => 'ID',
			'order'       => 'ASC',
			'tax_query'   => array(
				array(
					'taxonomy' => 'entity',
					'field'    => 'term_id',
					'terms'    => $entity->term_id,
				),
			),
		)
	);
	if ( empty( $products ) ) {
		return null;
	}
	$deepest = null;
	$depth   = -1;
	foreach ( wp_get_post_terms( $products[0]->ID, 'product_cat' ) as $category ) {
		$level = count( get_ancestors( $category->term_id, 'product_cat', 'taxonomy' ) );
		if ( $level > $depth ) {
			$deepest = $category;
			$depth   = $level;
		}
	}
	return $deepest;
}

function shop_hub_trail( WP_Post $post, WP_Term $term ) {
	$crumbs   = array(
		array(
			'name' => 'Home',
			'url'  => home_url( '/' ),
		),
	);
	$category = 'product_cat' === $term->taxonomy ? $term : shop_entity_category( $term );
	if ( $category instanceof WP_Term ) {
		$chain   = array_reverse( get_ancestors( $category->term_id, 'product_cat', 'taxonomy' ) );
		$chain[] = $category->term_id;
		foreach ( $chain as $term_id ) {
			$link = get_term( $term_id, 'product_cat' );
			if ( $link instanceof WP_Term && $link->slug !== $post->post_name ) {
				$crumbs[] = array(
					'name' => $link->name,
					'url'  => home_url( '/' . $link->slug . '/' ),
				);
			}
		}
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
	$post = get_post();
	$term = shop_hub_term( $post );
	if ( null === $term ) {
		return $content;
	}
	return shop_breadcrumb_nav( shop_hub_trail( $post, $term ) ) . $content . shop_product_section( $term );
}

function shop_hub_schema() {
	if ( ! is_page() ) {
		return;
	}
	$post = get_queried_object();
	$term = shop_hub_term( $post );
	if ( null === $term ) {
		return;
	}
	$url      = get_permalink( $post );
	$elements = array();
	foreach ( shop_hub_trail( $post, $term ) as $index => $crumb ) {
		$elements[] = array(
			'@type'    => 'ListItem',
			'position' => $index + 1,
			'name'     => $crumb['name'],
			'item'     => $crumb['url'],
		);
	}
	$items = array();
	foreach ( shop_hub_products( $term ) as $index => $product ) {
		$items[] = array(
			'@type'    => 'ListItem',
			'position' => $index + 1,
			'name'     => get_the_title( $product ),
			'url'      => get_permalink( $product ),
		);
	}
	$graph = array(
		array(
			'@context'   => 'https://schema.org',
			'@type'      => 'CollectionPage',
			'@id'        => $url . '#collection',
			'url'        => $url,
			'name'       => get_the_title( $post ),
			'breadcrumb' => array( '@id' => $url . '#trail' ),
			'mainEntity' => array( '@id' => $url . '#listing' ),
		),
		array(
			'@context'        => 'https://schema.org',
			'@type'           => 'BreadcrumbList',
			'@id'             => $url . '#trail',
			'itemListElement' => $elements,
		),
		array(
			'@context'        => 'https://schema.org',
			'@type'           => 'ItemList',
			'@id'             => $url . '#listing',
			'numberOfItems'   => count( $items ),
			'itemListElement' => $items,
		),
	);
	foreach ( $graph as $node ) {
		echo '<script type="application/ld+json">' . wp_json_encode( $node ) . '</script>' . "\n";
	}
}
