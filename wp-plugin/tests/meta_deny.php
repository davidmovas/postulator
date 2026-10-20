<?php

define( 'ABSPATH', __DIR__ . '/' );

require __DIR__ . '/../postulator-companion/includes/meta.php';

$table = json_decode( (string) file_get_contents( __DIR__ . '/../testdata/meta_deny.json' ), true );
if ( ! is_array( $table ) || empty( $table['cases'] ) ) {
	fwrite( STDERR, "the meta deny table is missing or empty\n" );
	exit( 1 );
}

$failed = 0;
foreach ( $table['cases'] as $case ) {
	$got = \Postulator\Companion\meta_key_denied( $case['key'] );
	if ( $got !== $case['denied'] ) {
		fwrite( STDERR, sprintf( "meta_key_denied(%s): got %s, want %s\n", json_encode( $case['key'] ), json_encode( $got ), json_encode( $case['denied'] ) ) );
		++$failed;
	}
}

$newline = \Postulator\Companion\meta_key_denied( "subtitle\n" );
if ( true !== $newline ) {
	fwrite( STDERR, "meta_key_denied(\"subtitle\\n\"): a key with a trailing newline is allowed\n" );
	++$failed;
}

$total = count( $table['cases'] ) + 1;
if ( $failed > 0 ) {
	fwrite( STDERR, sprintf( "%d of %d cases failed\n", $failed, $total ) );
	exit( 1 );
}
printf( "%d cases agree with field.MetaKeyAllowed\n", $total );
