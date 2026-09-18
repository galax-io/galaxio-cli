package main

import "github.com/spf13/cobra"

// outputModes is what an invocation may draw. Each rule is decided once, so
// that the summary on standard output and the progress block on standard error
// cannot disagree about a terminal or about --no-color.
type outputModes struct {
	// Color says the summary is drawn in colour.
	Color bool
	// Block says the progress block is drawn while the log is read.
	Block bool
	// BlockColor says the block is drawn in colour.
	BlockColor bool
}

// streams is what an invocation's two output streams can do and what its flags
// allow: colour is not switched off, --quiet was not given, and each stream
// understands escape sequences.
type streams struct {
	color, loud, stdout, stderr bool
}

// modes is what those streams may be drawn: colour needs the stream to
// understand escape sequences and colour not to be switched off, and the block
// needs standard error to be such a stream and --quiet not to be given, because
// the block is the one diagnostic a quiet run does not want redrawn at it.
func (s streams) modes() outputModes {
	return outputModes{
		Color:      s.color && s.stdout,
		Block:      s.stderr && s.loud,
		BlockColor: s.color && s.stderr,
	}
}

// modesFor reads the modes of cmd. It is the wiring; streams.modes is the rule.
func modesFor(cmd *cobra.Command) outputModes {
	return streams{
		color:  !isNoColor(cmd),
		loud:   !isQuiet(cmd),
		stdout: ansiTerminal(cmd.OutOrStdout()),
		stderr: ansiTerminal(cmd.ErrOrStderr()),
	}.modes()
}
