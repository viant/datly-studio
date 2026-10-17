# Reusable Studio API startup

`studioapi.Run` starts the same SDK/browser host as `cmd/studio-api`, with the
same command-line flags, authentication, namespace routing and shutdown behavior.
Custom applications can link their predicates before calling it:

```go
package main

import (
    _ "example.com/application/predicate"
    "github.com/viant/datly-studio/app/studioapi"
)

func main() { studioapi.Run() }
```

Select linked predicates with the existing `STUDIO_PREDICATE_PACKAGES` deployment
configuration. Runtime dynamic readers and native authoring components must also
link the selected packages in their respective binaries. Linking a handler does
not provision its Studio catalog entry or grant access to a resource.

Call `Run` once from a command's main function. It owns the process flag set and
signal lifecycle; SDK HTTP/MCP endpoints remain implemented by Datly components.

### Embedding the HTTP host

`studioapi.Serve(ctx, args, options) error` runs the HTTP host with a caller-owned
lifecycle. Arguments exclude the executable name. Each call uses an independent
flag set; it does not change `os.Args` or `flag.CommandLine`. Startup failures
return to the caller and release resources already opened. Cancellation drains
HTTP requests before closing dependencies. `--help` returns `flag.ErrHelp`.

`RunWithOptions` remains the standalone CLI wrapper: it installs signal handling
and reports terminal errors. Embedding applications should use `Serve` so failures
can trigger cleanup of their other listeners.

This API is a prerequisite for the shared single-process host, not the complete
unified launcher. The target default deployment is one binary and one shared
configuration for HTTP, admin MCP and default namespace MCP; separate service
startup is explicit opt-in. The shared configuration must reuse Datly connectors
(including SCY `Secret`) and `JWTValidator` (`RSA` or `CertURL`). Namespace and
publication definitions remain database-owned. Standard E2E must use the unified
single-process deployment before that deployment can be considered accepted.

A parent can pass `Options.Database` and `Options.DatabaseDriver` to reuse a
catalog connection opened through Datly's named connector and SCY configuration.
The HTTP host borrows this handle and never closes it, including on startup
failure. The parent must close it after all dependent services stop. Standalone
startup still opens and closes its own catalog connection. Development clone
operations still require a matching DSN; this does not yet remove all DSN-based
runtime adapters.

`Options.JWTValidator` accepts the shared SCY verifier configuration directly,
including multiple RSA resources and encryption keys, or `CertURL`. When supplied,
certificate/key CLI or environment overrides are rejected to avoid two competing
verification configurations. Issuer, audience, session key and browser login
validation continue to apply. The HTTP host currently supports RSA or certificate
verification, matching its existing authentication contract.
