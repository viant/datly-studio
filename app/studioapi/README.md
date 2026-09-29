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
