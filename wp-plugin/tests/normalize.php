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
	$got                 = \Postulator\Companion\internal_href_to_path( $case['href'], '/' );
	$listed              = 'path' === $case['kind'];
	$relative            = ! empty( $case['relative'] );

	if ( ( '' !== $got ) !== $listed ) {
		fwrite( STDERR, sprintf( "%s %s: got %s, want the kind %s\n", $case['site'], json_encode( $case['href'] ), json_encode( $got ), $case['kind'] ) );
		++$failed;
		continue;
	}
	if ( $listed && ! $relative && $got !== $case['path'] ) {
		fwrite( STDERR, sprintf( "%s %s: got %s, want %s\n", $case['site'], json_encode( $case['href'] ), json_encode( $got ), json_encode( $case['path'] ) ) );
		++$failed;
	}
}

if ( $failed > 0 ) {
	fwrite( STDERR, sprintf( "%d of %d cases failed\n", $failed, count( $table['cases'] ) ) );
	exit( 1 );
}
printf( "%d cases agree with Site.Resolve\n", count( $table['cases'] ) );
