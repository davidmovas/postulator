<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function permission_check() {
	if ( current_user_can( 'edit_posts' ) ) {
		return true;
	}
	return forbidden( 'an application password for a user with edit_posts is required' );
}

function forbidden( string $message ): \WP_Error {
	return new \WP_Error( 'forbidden', $message, array( 'status' => rest_authorization_required_code() ) );
}

function not_found( string $message ): \WP_Error {
	return new \WP_Error( 'not_found', $message, array( 'status' => 404 ) );
}

function invalid( string $code, string $message ): \WP_Error {
	return new \WP_Error( $code, $message, array( 'status' => 400 ) );
}

function failed( string $message ): \WP_Error {
	return new \WP_Error( 'update_failed', $message, array( 'status' => 500 ) );
}
