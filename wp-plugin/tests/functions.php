<?php

namespace PluginScan;

require __DIR__ . '/strict.php';

const HOOK_CALLERS       = array( 'add_action', 'add_filter', 'remove_action', 'remove_filter', 'has_action', 'has_filter', 'apply_filters', 'apply_filters_ref_array', 'do_action', 'do_action_ref_array' );
const CALLBACK_ARGUMENTS = array(
	'add_action'                 => 1,
	'add_filter'                 => 1,
	'remove_action'              => 1,
	'remove_filter'              => 1,
	'has_action'                 => 1,
	'has_filter'                 => 1,
	'array_map'                  => 0,
	'array_filter'               => 1,
	'array_walk'                 => 1,
	'usort'                      => 1,
	'uasort'                     => 1,
	'uksort'                     => 1,
	'preg_replace_callback'      => 1,
	'call_user_func'             => 0,
	'call_user_func_array'       => 0,
	'register_shutdown_function' => 0,
);
const BUILTIN_TYPES      = array( 'array', 'bool', 'callable', 'false', 'float', 'int', 'iterable', 'mixed', 'never', 'null', 'object', 'parent', 'self', 'static', 'string', 'true', 'void' );
const LITERALS           = array( 'true', 'false', 'null' );
const KINDS              = array(
	'functions' => 'function',
	'hooks'     => 'hook',
	'classes'   => 'class',
	'constants' => 'constant',
);

function token_id( string $name ): int {
	return defined( $name ) ? (int) constant( $name ) : -1;
}

function name_tokens(): array {
	return array( T_STRING, token_id( 'T_NAME_QUALIFIED' ), token_id( 'T_NAME_FULLY_QUALIFIED' ), token_id( 'T_NAME_RELATIVE' ) );
}

function member_operators(): array {
	return array( T_OBJECT_OPERATOR, T_DOUBLE_COLON, token_id( 'T_NULLSAFE_OBJECT_OPERATOR' ) );
}

function tokens_of( string $path ): array {
	$raw     = token_get_all( (string) file_get_contents( $path ) );
	$skipped = array( T_WHITESPACE, T_COMMENT, T_DOC_COMMENT, T_OPEN_TAG, T_CLOSE_TAG );
	$names   = name_tokens();
	$tokens  = array();
	$count   = count( $raw );

	for ( $i = 0; $i < $count; $i++ ) {
		$token = $raw[ $i ];
		if ( ! is_array( $token ) ) {
			$tokens[] = array(
				'kind' => $token,
				'text' => $token,
				'line' => empty( $tokens ) ? 1 : $tokens[ count( $tokens ) - 1 ]['line'],
			);
			continue;
		}
		if ( in_array( $token[0], $skipped, true ) ) {
			continue;
		}
		$starts_name = in_array( $token[0], $names, true ) || ( T_NS_SEPARATOR === $token[0] && isset( $raw[ $i + 1 ] ) && is_array( $raw[ $i + 1 ] ) && T_STRING === $raw[ $i + 1 ][0] );
		if ( ! $starts_name ) {
			$tokens[] = array(
				'kind' => $token[0],
				'text' => $token[1],
				'line' => $token[2],
			);
			continue;
		}
		$text = $token[1];
		$line = $token[2];
		while ( isset( $raw[ $i + 1 ] ) && is_array( $raw[ $i + 1 ] ) && in_array( $raw[ $i + 1 ][0], array( T_NS_SEPARATOR, T_STRING ), true ) ) {
			$text .= $raw[ $i + 1 ][1];
			++$i;
		}
		$tokens[] = array(
			'kind' => 'name',
			'text' => $text,
			'line' => $line,
		);
	}
	return $tokens;
}

function kind_at( array $tokens, int $at ) {
	return isset( $tokens[ $at ] ) ? $tokens[ $at ]['kind'] : null;
}

function unquote( string $literal ): string {
	$body = substr( $literal, 1, -1 );
	if ( "'" === $literal[0] ) {
		return strtr(
			$body,
			array(
				'\\\\' => '\\',
				"\\'"  => "'",
			)
		);
	}
	return stripcslashes( $body );
}

function qualified( string $namespace, string $name ): string {
	return strtolower( '' === $namespace ? $name : $namespace . '\\' . $name );
}

function arguments( array $tokens, int $open ): array {
	$opening   = array( '(', '[', '{', T_CURLY_OPEN, T_DOLLAR_OPEN_CURLY_BRACES );
	$closing   = array( ')', ']', '}' );
	$arguments = array();
	$current   = array();
	$depth     = 0;
	$count     = count( $tokens );

	for ( $i = $open + 1; $i < $count; $i++ ) {
		$kind = $tokens[ $i ]['kind'];
		if ( 0 === $depth && ( ')' === $kind || ',' === $kind ) ) {
			if ( ! empty( $current ) ) {
				$arguments[] = $current;
			}
			if ( ')' === $kind ) {
				return $arguments;
			}
			$current = array();
			continue;
		}
		if ( in_array( $kind, $opening, true ) ) {
			++$depth;
		} elseif ( in_array( $kind, $closing, true ) ) {
			--$depth;
		}
		$current[] = $tokens[ $i ];
	}
	return $arguments;
}

function single_literal( array $argument ): ?string {
	if ( 1 === count( $argument ) && T_CONSTANT_ENCAPSED_STRING === $argument[0]['kind'] ) {
		return unquote( $argument[0]['text'] );
	}
	return null;
}

final class Scan {
	public $root;
	public $files      = array();
	public $functions  = array();
	public $constants  = array();
	public $classes    = array();
	public $uses       = array(
		'functions' => array(),
		'hooks'     => array(),
		'classes'   => array(),
		'constants' => array(),
	);
	public $calls      = array();
	public $deferred   = array();
	public $failures   = array();

	public function __construct( string $root ) {
		$this->root = $root;
		$found      = new \RecursiveIteratorIterator( new \RecursiveDirectoryIterator( $root, \FilesystemIterator::SKIP_DOTS ) );
		foreach ( $found as $file ) {
			if ( 'php' === strtolower( $file->getExtension() ) ) {
				$this->files[ substr( str_replace( '\\', '/', $file->getPathname() ), strlen( $root ) + 1 ) ] = tokens_of( $file->getPathname() );
			}
		}
		ksort( $this->files );
	}

	public function fail( string $where, string $message ): void {
		$this->failures[] = $where . ': ' . $message;
	}

	public function use_of( string $kind, string $name, string $where ): void {
		$this->uses[ $kind ][ $name ][] = $where;
	}

	public function declare_all(): void {
		foreach ( $this->files as $tokens ) {
			$namespace = '';
			foreach ( $tokens as $at => $token ) {
				if ( 'name' !== $token['kind'] ) {
					continue;
				}
				$before = kind_at( $tokens, $at - 1 );
				if ( T_NAMESPACE === $before ) {
					$namespace = $token['text'];
				} elseif ( T_FUNCTION === $before ) {
					$this->functions[ qualified( $namespace, $token['text'] ) ] = true;
				} elseif ( T_CONST === $before ) {
					$this->constants[ ( '' === $namespace ? '' : strtolower( $namespace ) . '\\' ) . $token['text'] ] = true;
				} elseif ( in_array( $before, array( T_CLASS, T_INTERFACE, T_TRAIT ), true ) ) {
					$this->classes[ qualified( $namespace, $token['text'] ) ] = true;
				}
			}
		}
	}

	public function read_all(): void {
		foreach ( $this->files as $file => $tokens ) {
			$this->read( $file, $tokens );
		}
		foreach ( $this->deferred as $deferred ) {
			$this->resolve_deferred( $deferred );
		}
	}

	private function read( string $file, array $tokens ): void {
		$namespace = '';
		$in_string = false;
		$signature = null;
		$frames    = array();
		$depth     = 0;

		foreach ( $tokens as $at => $token ) {
			$kind  = $token['kind'];
			$where = $file . ':' . $token['line'];

			if ( '"' === $kind || '`' === $kind ) {
				$in_string = ! $in_string;
				continue;
			}
			if ( T_START_HEREDOC === $kind || T_END_HEREDOC === $kind ) {
				$in_string = T_START_HEREDOC === $kind;
				continue;
			}
			if ( T_FUNCTION === $kind && null === $signature ) {
				$named     = 'name' === kind_at( $tokens, $at + 1 );
				$signature = array(
					'name'   => $named ? qualified( $namespace, $tokens[ $at + 1 ]['text'] ) : null,
					'params' => array(),
					'parens' => 0,
					'open'   => true,
				);
				continue;
			}
			if ( null !== $signature ) {
				if ( '(' === $kind ) {
					++$signature['parens'];
				} elseif ( ')' === $kind ) {
					--$signature['parens'];
					if ( 0 === $signature['parens'] ) {
						$signature['open'] = false;
					}
				} elseif ( T_VARIABLE === $kind && $signature['open'] && 1 === $signature['parens'] ) {
					$signature['params'][] = $token['text'];
				} elseif ( '{' === $kind ) {
					$frames[]  = array(
						'depth'  => $depth,
						'name'   => $signature['name'],
						'params' => $signature['params'],
					);
					$signature = null;
					++$depth;
					continue;
				} elseif ( ';' === $kind ) {
					$signature = null;
					continue;
				}
			}
			if ( in_array( $kind, array( '{', T_CURLY_OPEN, T_DOLLAR_OPEN_CURLY_BRACES ), true ) ) {
				++$depth;
				continue;
			}
			if ( '}' === $kind ) {
				--$depth;
				if ( ! empty( $frames ) && end( $frames )['depth'] === $depth ) {
					array_pop( $frames );
				}
				continue;
			}
			if ( T_NS_C === $kind && '.' === kind_at( $tokens, $at + 1 ) && T_CONSTANT_ENCAPSED_STRING === kind_at( $tokens, $at + 2 ) ) {
				$named = strtolower( $namespace . unquote( $tokens[ $at + 2 ]['text'] ) );
				if ( ! isset( $this->functions[ $named ] ) ) {
					$this->fail( $where, sprintf( '%s names no function the plugin defines', $named ) );
				}
				continue;
			}
			if ( 'name' !== $kind || $in_string ) {
				continue;
			}

			$before = kind_at( $tokens, $at - 1 );
			$after  = kind_at( $tokens, $at + 1 );
			$frame  = empty( $frames ) ? null : end( $frames );

			if ( in_array( $before, member_operators(), true ) || in_array( $before, array( T_FUNCTION, T_CONST, T_CLASS, T_INTERFACE, T_TRAIT ), true ) ) {
				continue;
			}
			if ( T_NAMESPACE === $before ) {
				$namespace = $token['text'];
				continue;
			}
			if ( T_USE === $before ) {
				$this->fail( $where, sprintf( 'imports %s, and the scan reads no use statement', $token['text'] ) );
				continue;
			}
			if ( null !== $signature ) {
				if ( '=' === $before ) {
					$this->constant( $namespace, $token['text'], $where );
				} else {
					$this->type( $namespace, $token['text'], $where );
				}
				continue;
			}
			if ( in_array( $before, array( T_NEW, T_INSTANCEOF, T_EXTENDS, T_IMPLEMENTS ), true ) || in_array( $after, array( T_DOUBLE_COLON, T_VARIABLE ), true ) ) {
				$this->type( $namespace, $token['text'], $where );
				continue;
			}
			if ( '(' === $after ) {
				$this->call( $namespace, $token['text'], arguments( $tokens, $at + 1 ), $frame, $where );
				continue;
			}
			$this->constant( $namespace, $token['text'], $where );
		}
	}

	private function type( string $namespace, string $name, string $where ): void {
		if ( '\\' !== $name[0] && in_array( strtolower( $name ), BUILTIN_TYPES, true ) ) {
			return;
		}
		if ( '\\' === $name[0] ) {
			$global = substr( $name, 1 );
		} elseif ( '' === $namespace ) {
			$global = $name;
		} else {
			$global = $namespace . '\\' . $name;
		}
		if ( isset( $this->classes[ strtolower( $global ) ] ) ) {
			return;
		}
		if ( false !== strpos( $global, '\\' ) ) {
			$this->fail( $where, sprintf( 'names the class %s, which the plugin does not declare', $global ) );
			return;
		}
		$this->use_of( 'classes', $global, $where );
	}

	private function constant( string $namespace, string $name, string $where ): void {
		if ( in_array( strtolower( $name ), LITERALS, true ) ) {
			return;
		}
		if ( '\\' !== $name[0] && false === strpos( $name, '\\' ) && '' !== $namespace && isset( $this->constants[ strtolower( $namespace ) . '\\' . $name ] ) ) {
			return;
		}
		$global = ltrim( $name, '\\' );
		if ( isset( $this->constants[ $global ] ) ) {
			return;
		}
		if ( false !== strpos( $global, '\\' ) ) {
			$this->fail( $where, sprintf( 'reads the constant %s, which the plugin does not declare', $global ) );
			return;
		}
		$this->use_of( 'constants', $global, $where );
	}

	private function function_name( string $namespace, string $name ): array {
		if ( '\\' === $name[0] ) {
			$global = strtolower( substr( $name, 1 ) );
			return array( isset( $this->functions[ $global ] ), $global );
		}
		if ( false !== strpos( $name, '\\' ) ) {
			$qualified = qualified( $namespace, $name );
			return array( isset( $this->functions[ $qualified ] ), $qualified );
		}
		$own = qualified( $namespace, $name );
		if ( isset( $this->functions[ $own ] ) ) {
			return array( true, $own );
		}
		return array( isset( $this->functions[ strtolower( $name ) ] ), strtolower( $name ) );
	}

	private function call( string $namespace, string $name, array $arguments, ?array $frame, string $where ): void {
		list( $own, $resolved ) = $this->function_name( $namespace, $name );
		if ( $own ) {
			$this->calls[ $resolved ][] = array(
				'arguments' => $arguments,
				'where'     => $where,
			);
			return;
		}
		if ( false !== strpos( $resolved, '\\' ) ) {
			$this->fail( $where, sprintf( 'calls %s, which the plugin does not define', $resolved ) );
			return;
		}
		$this->use_of( 'functions', $resolved, $where );

		if ( in_array( $resolved, HOOK_CALLERS, true ) ) {
			$this->hook( isset( $arguments[0] ) ? $arguments[0] : array(), $frame, $resolved, $where );
		}
		if ( array_key_exists( $resolved, CALLBACK_ARGUMENTS ) && isset( $arguments[ CALLBACK_ARGUMENTS[ $resolved ] ] ) ) {
			$this->callback( $arguments[ CALLBACK_ARGUMENTS[ $resolved ] ], $where );
		}
	}

	private function hook( array $argument, ?array $frame, string $caller, string $where ): void {
		$literal = single_literal( $argument );
		if ( null !== $literal ) {
			$this->use_of( 'hooks', $literal, $where );
			return;
		}
		if ( 1 === count( $argument ) && T_VARIABLE === $argument[0]['kind'] && null !== $frame && null !== $frame['name'] ) {
			$position = array_search( $argument[0]['text'], $frame['params'], true );
			if ( false !== $position ) {
				$this->deferred[] = array(
					'function' => $frame['name'],
					'position' => $position,
					'where'    => $where,
				);
				return;
			}
		}
		$this->fail( $where, sprintf( '%s takes its hook from an expression the scan cannot follow', $caller ) );
	}

	private function callback( array $argument, string $where ): void {
		$literal = single_literal( $argument );
		if ( null === $literal ) {
			return;
		}
		if ( false !== strpos( $literal, '::' ) ) {
			$this->fail( $where, sprintf( 'passes the method %s as a callback, and the scan follows functions only', $literal ) );
			return;
		}
		$named = strtolower( ltrim( $literal, '\\' ) );
		if ( isset( $this->functions[ $named ] ) ) {
			return;
		}
		if ( false !== strpos( $named, '\\' ) ) {
			$this->fail( $where, sprintf( 'passes %s as a callback, which the plugin does not define', $named ) );
			return;
		}
		$this->use_of( 'functions', $named, $where );
	}

	private function resolve_deferred( array $deferred ): void {
		if ( empty( $this->calls[ $deferred['function'] ] ) ) {
			$this->fail( $deferred['where'], sprintf( 'reads its hook from a parameter of %s, which nothing calls', $deferred['function'] ) );
			return;
		}
		foreach ( $this->calls[ $deferred['function'] ] as $call ) {
			$argument = isset( $call['arguments'][ $deferred['position'] ] ) ? $call['arguments'][ $deferred['position'] ] : array();
			$literal  = single_literal( $argument );
			if ( null === $literal ) {
				$this->fail( $call['where'], sprintf( 'passes %s a hook the scan cannot read', $deferred['function'] ) );
				continue;
			}
			$this->use_of( 'hooks', $literal, $call['where'] );
		}
	}
}

function folded_keys( array $map ): array {
	return array_change_key_case( $map, CASE_LOWER );
}

function present_in_php( string $kind, string $name ): bool {
	switch ( $kind ) {
		case 'functions':
			return function_exists( $name );
		case 'classes':
			return class_exists( $name ) || interface_exists( $name );
		case 'constants':
			return defined( $name );
	}
	return false;
}

function trimmed_version( string $version ): string {
	return (string) preg_replace( '/(\.0)+$/', '', $version );
}

$plugin_root = dirname( __DIR__ ) . '/postulator-companion';
$kept        = json_decode( (string) file_get_contents( dirname( __DIR__ ) . '/testdata/wordpress.json' ), true );
if ( ! is_array( $kept ) || ! isset( $kept['wordpress'], $kept['thirdParty'] ) ) {
	fwrite( STDERR, "the kept list of WordPress and third-party names is missing or unreadable\n" );
	exit( 1 );
}

$scan = new Scan( $plugin_root );
if ( empty( $scan->files ) ) {
	fwrite( STDERR, "the plugin holds no PHP file\n" );
	exit( 1 );
}
$scan->declare_all();
$scan->read_all();

$newest      = array(
	'name'  => '',
	'since' => '0',
);
$sensitive   = array( 'hooks', 'constants' );
$used_counts = array();
foreach ( KINDS as $kind => $singular ) {
	$wordpress = isset( $kept['wordpress'][ $kind ] ) ? $kept['wordpress'][ $kind ] : array();
	$vendors   = isset( $kept['thirdParty'][ $kind ] ) ? $kept['thirdParty'][ $kind ] : array();
	if ( ! in_array( $kind, $sensitive, true ) ) {
		$wordpress = folded_keys( $wordpress );
		$vendors   = folded_keys( $vendors );
	}
	$used = array();
	foreach ( $scan->uses[ $kind ] as $name => $wheres ) {
		$key          = in_array( $kind, $sensitive, true ) ? $name : strtolower( $name );
		$used[ $key ] = true;
		if ( isset( $wordpress[ $key ] ) ) {
			if ( 1 !== preg_match( '/^\d+\.\d+(\.\d+)?$/', (string) $wordpress[ $key ] ) ) {
				$scan->fail( 'testdata/wordpress.json', sprintf( 'the %s %s has no readable @since: %s', $singular, $name, json_encode( $wordpress[ $key ] ) ) );
				continue;
			}
			if ( version_compare( (string) $wordpress[ $key ], $newest['since'], '>' ) ) {
				$newest = array(
					'name'  => $name,
					'since' => (string) $wordpress[ $key ],
				);
			}
			continue;
		}
		if ( isset( $vendors[ $key ] ) ) {
			continue;
		}
		if ( 'hooks' !== $kind && present_in_php( $kind, $name ) ) {
			continue;
		}
		$scan->fail( $wheres[0], sprintf( 'uses the %s %s, which is neither the plugin\'s, on the kept list nor in PHP %s', $singular, $name, PHP_VERSION ) );
	}
	foreach ( array_merge( array_keys( $wordpress ), array_keys( $vendors ) ) as $listed ) {
		if ( ! isset( $used[ $listed ] ) ) {
			$scan->fail( 'testdata/wordpress.json', sprintf( 'the kept list names the %s %s, which the plugin no longer uses', $singular, $listed ) );
		}
	}
	$used_counts[ $kind ] = count( $used );
}

$header = (string) file_get_contents( $plugin_root . '/postulator-companion.php' );
if ( 1 !== preg_match( '/^\s*\*\s*Requires at least:\s*(\S+)\s*$/m', $header, $declared ) ) {
	$scan->fail( 'postulator-companion.php', 'the header declares no Requires at least' );
} elseif ( trimmed_version( $declared[1] ) !== trimmed_version( $newest['since'] ) ) {
	$scan->fail( 'postulator-companion.php', sprintf( 'the header requires WordPress %s, and the newest name the plugin uses, %s, is there since %s', $declared[1], $newest['name'], $newest['since'] ) );
}

if ( ! empty( $scan->failures ) ) {
	foreach ( $scan->failures as $failure ) {
		fwrite( STDERR, $failure . "\n" );
	}
	fwrite( STDERR, sprintf( "the plugin's names fail %d checks on PHP %s\n", count( $scan->failures ), PHP_VERSION ) );
	exit( 1 );
}
printf(
	"%d files: %d functions, %d hooks, %d classes and %d constants are the plugin's, kept or PHP %s's; the newest, %s, is there since WordPress %s\n",
	count( $scan->files ),
	$used_counts['functions'],
	$used_counts['hooks'],
	$used_counts['classes'],
	$used_counts['constants'],
	PHP_VERSION,
	$newest['name'],
	$newest['since']
);
