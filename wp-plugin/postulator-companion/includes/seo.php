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
