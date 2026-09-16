package output

// ResolveColor decides whether colored output should be produced.
//
// Precedence, highest to lowest:
//
//  1. requestedExplicit: the CLI's --color/--no-color flag was passed
//     explicitly, so honor it verbatim, whatever the environment says.
//  2. noColorEnv: the NO_COLOR convention (https://no-color.org). Any
//     non-empty value disables color, regardless of the terminal.
//  3. forceColorEnv: any non-empty FORCE_COLOR value enables color, letting
//     users get colored output through a pipe (e.g. `| less -R`).
//  4. stdoutIsTerminal: fall back to auto-detection, the sane default for
//     both interactive terminals and piped/redirected output.
//
// Callers resolve this once per invocation and pass the result down as an
// explicit parameter; it must not be cached in a package-level variable.
func ResolveColor(requested, requestedExplicit bool, noColorEnv, forceColorEnv string, stdoutIsTerminal bool) bool {
	if requestedExplicit {
		return requested
	}
	if noColorEnv != "" {
		return false
	}
	if forceColorEnv != "" {
		return true
	}
	return stdoutIsTerminal
}
