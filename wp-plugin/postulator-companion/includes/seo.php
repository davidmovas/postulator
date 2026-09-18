<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const META_KEYS = array(
	'yoast'    => array(
		'title'         => '_yoast_wpseo_title',
		'description'   => '_yoast_wpseo_metadesc',
		'canonical'     => '_yoast_wpseo_canonical',
		'ogTitle'       => '_yoast_wpseo_opengraph-title',
		'ogDescription' => '_yoast_wpseo_opengraph-description',
	),
	'rankmath' => array(
		'title'         => 'rank_math_title',
		'description'   => 'rank_math_description',
		'canonical'     => 'rank_math_canonical_url',
		'ogTitle'       => 'rank_math_facebook_title',
		'ogDescription' => 'rank_math_facebook_description',
	),
	'none'     => array(
		'title'         => '_postulator_seo_title',
		'description'   => '_postulator_seo_description',
		'canonical'     => '_postulator_canonical',
		'ogTitle'       => '_postulator_og_title',
		'ogDescription' => '_postulator_og_description',
	),
);

function detect_plugin(): string {
	if ( defined( 'WPSEO_VERSION' ) ) {
		return 'yoast';
	}
	if ( class_exists( 'RankMath' ) ) {
		return 'rankmath';
	}
	return 'none';
}

function read_post_seo( int $post_id ): array {
	$map = META_KEYS[ detect_plugin() ];

	return array(
		'title'       => (string) get_post_meta( $post_id, $map['title'], true ),
		'description' => (string) get_post_meta( $post_id, $map['description'], true ),
		'canonical'   => (string) get_post_meta( $post_id, $map['canonical'], true ),
	);
}

function read_term_seo( \WP_Term $term ): array {
	$plugin = detect_plugin();

	if ( 'yoast' === $plugin ) {
		$all = get_option( 'wpseo_taxonomy_meta', array() );
		$row = is_array( $all ) && isset( $all[ $term->taxonomy ][ $term->term_id ] ) ? $all[ $term->taxonomy ][ $term->term_id ] : array();

		return array(
			'title'       => isset( $row['wpseo_title'] ) ? (string) $row['wpseo_title'] : '',
			'description' => isset( $row['wpseo_desc'] ) ? (string) $row['wpseo_desc'] : '',
			'canonical'   => isset( $row['wpseo_canonical'] ) ? (string) $row['wpseo_canonical'] : '',
		);
	}

	$map = META_KEYS[ $plugin ];
	return array(
		'title'       => (string) get_term_meta( $term->term_id, $map['title'], true ),
		'description' => (string) get_term_meta( $term->term_id, $map['description'], true ),
		'canonical'   => (string) get_term_meta( $term->term_id, $map['canonical'], true ),
	);
}

function write_post_seo( int $post_id, array $fields ): array {
	$map     = META_KEYS[ detect_plugin() ];
	$applied = array();

	foreach ( $map as $field => $key ) {
		if ( ! array_key_exists( $field, $fields ) ) {
			continue;
		}
		$value = $fields[ $field ];
		if ( '' === $value ) {
			delete_post_meta( $post_id, $key );
		} else {
			update_post_meta( $post_id, $key, wp_slash( $value ) );
		}
		$applied[] = $field;
	}
	return $applied;
}
