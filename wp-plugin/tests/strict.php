<?php

error_reporting( E_ALL );

set_error_handler(
	function ( int $severity, string $message, string $file, int $line ): bool {
		fwrite( STDERR, sprintf( "%s:%d: PHP %s raised level %d: %s\n", $file, $line, PHP_VERSION, $severity, $message ) );
		exit( 1 );
	}
);
