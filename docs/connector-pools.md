# Explicit connection pool limits

Connector `options` may contain a `pool` object:

```json
{"pool":{"maxOpenConns":1,"maxIdleConns":1}}
```

These are `database/sql` connection limits, applied to the connector's shared
runtime pool. They are useful for services that require serialized connection
use. Omitted fields preserve the existing settings. An explicit `maxOpenConns`
of zero means unlimited connections; `maxIdleConns` zero disables idle retention.
Negative values are rejected. When both values are supplied with a positive open
limit, the idle limit must not exceed it.

Changing connection options uses the normal connector edit, connectivity test,
activation and namespace reload flow. No database migration is needed. Existing
connectors without a `pool` object retain their current behavior. SQLite schema
attachments continue to require their existing single-connection setup.
