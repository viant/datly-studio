# Release reconciliation

Release channels and pending operations are durable in Studio SQL. Each producer
process owns its own in-memory published manifest. A producer must call
`Controller.Recover` before serving, then run `Controller.RunSync` in a
background worker to learn about committed changes made by another process.
`RunSync` checks committed channel revisions, current named-policy dependencies,
and source availability. It never runs on a window or data request. Pending
operations are reserved for startup recovery so a second process cannot race
the writer while its projection is in progress.

The stock runtime host starts this worker for its configured release controller
and stops it before closing SQL storage. `Config.ReleaseSyncInterval` is a trusted
Go setting; zero selects a two-second interval. Pass a cancellable host context
when running a controller in another embedding application. An unchanged state
does not rebuild the manifest or notify MCP sessions. An unexpected store or
source error blocks release reads until reconciliation succeeds. Successful
changes emit the existing `resources/list_changed` notification through the
process-local published catalog; clients of stateless MCP transports still
need their normal background conditional synchronization.
