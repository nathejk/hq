# 107 — Graceful shutdown panics instead of exiting cleanly

**Status:** done
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Every clean stop of the API ends in a segfault.

`cmd/api/main.go` finishes with:

```go
logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)
```

`Serve` returns **nil** when a SIGINT/SIGTERM shutdown completes without error — that is its
documented success path, and the comment in `cmd/api/app/server.go` says so. `PrintFatal` then calls
`err.Error()` on a nil error:

```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation]
nathejk.dk/internal/jsonlog.(*Logger).PrintFatal(…)
	/app/internal/jsonlog/jsonlog.go:62
main.main()
	/app/cmd/api/main.go:341
exit status 2
```

Found while reading the api logs during PRD 008 work; 187 occurrences in one session, because the
dev container's hot-reload restarts the API on every file change. It is harmless to the running
service — the shutdown has already completed by then — but it means **every ordinary stop looks
like a crash**, which is exactly the noise that hides a real one. It also exits 2 where it should
exit 0, so any supervisor or CI step that checks the exit code learns the wrong thing.

Two changes, and both are wanted:

1. **The call site.** Only report a failure when there is one, and log the clean stop at INFO. That
   is the root cause: `PrintFatal` means "we are dying because of this error", and a successful
   shutdown is not that.
2. **The logger.** `PrintFatal(nil, …)` and `PrintError(nil, …)` should not crash. A logger that
   segfaults while reporting a problem turns one bug into a nil-pointer trace that hides it — and
   `PrintError` is reachable with a nil error from any handler that forgets a check. Guard it once,
   in the shared path, rather than at every call site.

Not a behaviour change anybody depends on: nothing can currently observe a clean exit, because
there isn't one.

## Acceptance Criteria

- [x] A clean SIGTERM shutdown logs at INFO and exits 0, with no panic
- [x] A real listen error still logs at FATAL and exits non-zero
- [x] `PrintError(nil, …)` and `PrintFatal(nil, …)` do not panic
- [x] `jsonlog` has a test for the nil case
- [x] Verified against the running dev container: restart produces no panic

## Progress Log

- 2026-08-23 22:20 — Task created and picked up. Found while checking api logs in task 101; recorded
  there and raised here rather than fixed in passing, because it is in the composition root.
- 2026-08-23 22:30 — Call site fixed: `Serve` is only fatal when it returns an error, and a clean
  stop logs "Application stopped" at INFO. The comment there explains what the old line did and why
  it mattered, since the code now looks unremarkable and the reason it changed would otherwise be
  invisible.
- 2026-08-23 22:35 — Logger hardened too: `errorMessage()` in one shared place, so `PrintError(nil)`
  and `PrintFatal(nil)` cannot crash. Two reasons for doing both rather than only the call site: the
  call site was the root cause, but `PrintError` is reachable with a nil error from any handler that
  forgets a check, and a logger that segfaults *while reporting a problem* replaces the diagnosis
  with a nil-pointer trace from inside the logging call — the original problem is never printed at
  all.
- 2026-08-23 22:36 — The nil case logs "nil error logged: the caller reported a failure without one"
  rather than an empty message. A caller reaching there has a bug of its own, and an empty FATAL
  line would give nobody a thread to pull.
- 2026-08-23 22:40 — ✅ Verified against the running dev container. The hot-reload restart now logs
  `shutting down server {"signal":"interrupt"}` followed by `Application stopped`, and the panic
  count in the container log was **161 before the restart and 161 after** — previously every stop
  added one. `PrintFatal`'s exit path cannot be called from a test (it exits the process), so the
  test covers `errorMessage`, which is where the fix actually lives; 4 tests in a new
  `internal/jsonlog/jsonlog_test.go`, a package that had none.
- 2026-08-23 22:41 — Moving to done.
