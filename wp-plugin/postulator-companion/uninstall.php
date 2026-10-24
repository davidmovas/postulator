<?php

defined( 'WP_UNINSTALL_PLUGIN' ) || exit;

delete_metadata( 'post', 0, '_postulator_preview_hash', '', true );
delete_metadata( 'post', 0, '_postulator_preview_expires', '', true );
delete_metadata( 'term', 0, '_postulator_modified', '', true );
