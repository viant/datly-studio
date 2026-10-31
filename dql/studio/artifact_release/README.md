# Release-store graphs

These internal graphs are the authoritative source of generated release-store
components. They use the existing artifact release schema without adding tables.

Generate with the project's pinned, published Datly CLI. For example:

```sh
datly transcribe patch -dir . -schema -connector studio -driver sqlite3 \
  -dsn schema.db github.com/viant/datly-studio/dql/studio/artifact_release/stage
```

`schema.db` is a discovery database with the existing Studio schema. Use `patch`
for Stage/Complete/Fail, `post` for create_channel/put_bundle, and `get` for
snapshot/bundle/intent/channels/history. Repeat generation must produce identical
bytes. Do not edit the generated Go or SQL; change DQL and regenerate instead.
Application business rules belong in the separate `rules.go` lifecycle files.

Physical table sources mark writable nodes. Parenthesized table sources mark
read-only snapshots. The Stage/Complete/Fail graphs deliberately use narrow
physical projections so that reserving or completing an operation cannot rewrite
unrelated channel metadata or immutable bundle/binding data.
