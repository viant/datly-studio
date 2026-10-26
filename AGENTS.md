# Project instructions

## Go context convention

- When a function or method accepts `context.Context`, it must be the first parameter. This includes test helpers and internal APIs.
- Pass the caller’s context as the first argument. Context is required, never an optional or variadic trailing argument. Do not pass nil or create a replacement background context just to satisfy a signature.
