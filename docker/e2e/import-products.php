<?php

require_once ABSPATH . 'wp-admin/includes/class-wp-importer.php';
require_once WC_ABSPATH . 'includes/import/class-wc-product-csv-importer.php';

$sheet = isset( $args[0] ) ? $args[0] : '';
if ( '' === $sheet || ! is_readable( $sheet ) ) {
	WP_CLI::error( 'there is no products sheet at ' . $sheet );
}

$columns    = array(
	'Type'                  => 'type',
	'SKU'                   => 'sku',
	'Name'                  => 'name',
	'Published'             => 'published',
	'Visibility in catalog' => 'catalog_visibility',
	'Short description'     => 'short_description',
	'Description'           => 'description',
	'In stock?'             => 'stock_status',
	'Regular price'         => 'regular_price',
	'Categories'            => 'category_ids',
	'Parent'                => 'parent_id',
	'Entity'                => 'entity',
);
$attributes = array(
	'name'     => 'name',
	'value(s)' => 'value',
	'visible'  => 'visible',
	'global'   => 'taxonomy',
	'default'  => 'default',
);

$handle  = fopen( $sheet, 'r' );
$headers = fgetcsv( $handle, 0, ',', '"', "\0" );
fclose( $handle );

$mapping = array();
foreach ( $headers as $header ) {
	if ( isset( $columns[ $header ] ) ) {
		$mapping[ $header ] = $columns[ $header ];
		continue;
	}
	if ( preg_match( '/^Attribute (\d+) (name|value\(s\)|visible|global|default)$/', $header, $match ) ) {
		$mapping[ $header ] = 'attributes:' . $attributes[ $match[2] ] . $match[1];
		continue;
	}
	WP_CLI::error( $sheet . ' has a column the importer does not take: ' . $header );
}

if ( in_array( 'entity', $mapping, true ) && ! taxonomy_exists( 'entity' ) ) {
	WP_CLI::error( $sheet . ' names entities and the site registers no entity taxonomy' );
}

add_action(
	'woocommerce_product_import_inserted_product_object',
	function ( $product, $data ) {
		if ( empty( $data['entity'] ) ) {
			return;
		}
		$assigned = wp_set_object_terms( $product->get_id(), array_map( 'trim', explode( ',', $data['entity'] ) ), 'entity' );
		if ( is_wp_error( $assigned ) ) {
			WP_CLI::error( $assigned->get_error_message() );
		}
	},
	10,
	2
);

$importer = new WC_Product_CSV_Importer(
	$sheet,
	array(
		'mapping'          => $mapping,
		'parse'            => true,
		'update_existing'  => false,
		'prevent_timeouts' => false,
	)
);
$result   = $importer->import();

foreach ( $result['failed'] as $failure ) {
	WP_CLI::warning( $failure->get_error_message() );
}
if ( ! empty( $result['failed'] ) ) {
	WP_CLI::error( count( $result['failed'] ) . ' rows of ' . $sheet . ' failed to import' );
}

$imported = count( $result['imported'] ) + ( isset( $result['imported_variations'] ) ? count( $result['imported_variations'] ) : 0 );
WP_CLI::success( sprintf( '%d rows imported from %s, %d rows already in the store', $imported, $sheet, count( $result['skipped'] ) ) );
