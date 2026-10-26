# Project instructions

## Go context convention

- When a function or method accepts `context.Context`, it must be the first parameter. This includes test helpers and internal APIs.
- Pass the caller’s context as the first argument. Context is required, never an optional or variadic trailing argument. Do not pass nil or create a replacement background context just to satisfy a signature.

## Application-neutral runtime

- Keep Data Studio independent and application-neutral. Application-specific binary extensions are limited to custom predicates and custom codecs; do not link per-report runners, report IDs, field mappings, business formulas or subscription workflows into the upstream runtime.
- Application-specific orchestration belongs in the consuming application, not the upstream runtime.
- Author application readers, writers, source branches and calculations as managed assets using reusable Datly capabilities. Shared mechanisms and their contracts must work without any particular report or namespace.
- Keep public APIs, examples, fixtures, documentation and commit messages domain-neutral. Do not include customer/business identities, private application terminology, production captures, private hosts or personal workstation paths.
