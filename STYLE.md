# Code style

## Errors

Functions that can fail return an `error`. They never call `log.Fatal`, `os.Exit`,
or `panic` — that decision belongs to the caller, not the callee.

Wrap every error you pass upward with the action that failed, so the final
message reads as a trace from the outermost intent down to the root cause:

```go
if err := os.WriteFile(tmp, data, 0o644); err != nil {
    return fmt.Errorf("failed to write %s: %w", tmp, err)
}
```

- Always `%w`, never `%v` — wrapping keeps `errors.Is`/`errors.As` working.
- Phrase it `failed to <action>`, and name the thing acted on (path, URL, set ID).
- Never return a bare `err` from a call that can fail in more than one place;
  an unwrapped error arrives at the top with no idea where it came from.
- Root causes — a bad HTTP status, a violated invariant — are stated directly
  rather than prefixed with "failed to", since nothing is being wrapped.

`main` is the only place the program exits. It dispatches, collects the one
error, and prints it:

```go
if err != nil {
    log.Fatalf("rb %s: %v", cmd, err)
}
```

A full chain then looks like:

```
rb collection: failed to load catalog: cards/cards.json not found,
run `rb download-cards` first: open cards/cards.json: no such file or directory
```

Subcommands use `flag.ContinueOnError` and return parse failures, rather than
`flag.ExitOnError` which exits behind the caller's back. `flag.ErrHelp` is not
an error — return `nil` so `-h` exits 0.

## Comments

Comment why, not what. Explain decisions the code can't state itself:
a workaround for upstream behaviour, a non-obvious algorithm, a tradeoff.
Skip comments that restate the next line.
