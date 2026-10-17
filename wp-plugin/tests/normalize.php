<?php

define( 'ABSPATH', __DIR__ . '/' );

$GLOBALS['site_url'] = '';

function home_url() {
	return $GLOBALS['site_url'];
}

function wp_parse_url( $url, $component = -1 ) {
	return parse_url( $url, $component );
}

require __DIR__ . '/../postulator-companion/includes/normalize.php';

$table = json_decode( (string) file_get_contents( __DIR__ . '/../testdata/hrefs.json' ), true );
if ( ! is_array( $table ) || empty( $table['cases'] ) ) {
	fwrite( STDERR, "the href table is missing or empty\n" );
	exit( 1 );
}

$failed = 0;
foreach ( $table['cases'] as $case ) {
	$GLOBALS['site_url'] = $case['site'];
	$document            = isset( $case['document'] ) ? $case['document'] : '/';
	$got                 = \Postulator\Companion\internal_href_to_path( $case['href'], \Postulator\Companion\href_base( $document ) );
	$listed              = 'path' === $case['kind'];
	$where               = sprintf( '%s %s %s', $case['site'], $document, json_encode( $case['href'] ) );

	if ( ( '' !== $got ) !== $listed ) {
		fwrite( STDERR, sprintf( "%s: got %s, want the kind %s\n", $where, json_encode( $got ), $case['kind'] ) );
		++$failed;
		continue;
	}
	if ( $listed && $got !== $case['path'] ) {
		fwrite( STDERR, sprintf( "%s: got %s, want %s\n", $where, json_encode( $got ), json_encode( $case['path'] ) ) );
		++$failed;
	}
}

$bases = array(
	'/x/y/'                        => '/x/y/',
	'/x/y'                         => '/x/',
	''                             => '/',
	'http://example.test:8087'     => '/',
	'http://example.test:8087/a/b' => '/a/',
	'https://example.test/Café/'   => '/Caf%C3%A9/',
	'/a%zz/b'                      => '/',
);
foreach ( $bases as $url => $want ) {
	$got = \Postulator\Companion\href_base( $url );
	if ( $got !== $want ) {
		fwrite( STDERR, sprintf( "href_base(%s): got %s, want %s\n", json_encode( $url ), json_encode( $got ), json_encode( $want ) ) );
		++$failed;
	}
}

$total = count( $table['cases'] ) + count( $bases );
if ( $failed > 0 ) {
	fwrite( STDERR, sprintf( "%d of %d cases failed\n", $failed, $total ) );
	exit( 1 );
}
printf( "%d cases agree with Site.ResolveAt\n", $total );
