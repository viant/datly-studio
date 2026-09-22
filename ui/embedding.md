# Embedding Datly Studio

The UI package exposes a small SDK instead of requiring host projects to fork
the application shell.

```jsx
import React from 'react';
import { createStudioSDK, defineStudioExtension, mountStudio } from 'datly-studio-ui';

const sdk = createStudioSDK().register(defineStudioExtension({
  id: 'acme.audit',
  label: 'Audit',
  icon: 'history',
  order: 600,
  render: ({ api, subject, navigate, openComponent }) => (
    <AuditWorkspace api={api} subject={subject} onOpenComponent={openComponent}/>
  ),
}));

mountStudio(document.getElementById('root'), { config, sdk });
```

Extension IDs are canonical and globally unique. A workspace receives the
public Studio API, authenticated subject, navigation function, and component
launcher. Extensions do not reach into `StudioApp` state.

Go hosts register linked Datly authorization handlers explicitly:

```go
catalog, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{
    Alias: "acmeiam",
    Path:  "example.com/acme/iam/authorization",
    Types: []reflect.Type{
        reflect.TypeFor[authorization.CustomerRead](),
    },
}}}).PredicateCatalog()
```

Only registered package/type pairs appear in the Security workspace and may be
activated in `authorization_predicates`.
