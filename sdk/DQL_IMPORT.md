# Loading DQL

## Downloading a component

```go
download, err := client.Versions().Download(ctx, reportID, versionNo)
// download.Archive contains ZIP bytes; download.Filename is a suggested name.
```

The archive contains `component.dql` (or a collision-free variant) at its root,
delegated view queries under `sql/`, and the version's stored dependency files.
Datly Reader Builder source spans drive SQL extraction. Existing embedded
resource references keep their paths. `EntryDQL` identifies the generated entry
document for a subsequent `LoadArchive` call. The operation requires DQL access
and does not change the component. HTTP clients call `versions.download` with
`reportId` and `versionNo`; the JSON response encodes ZIP bytes as base64.

Use an existing component's report ID to create a new draft. Imports require
both edit and DQL permissions. They never publish or execute uploaded source.

```go
result, err := client.Versions().LoadDQL(ctx, reportID, sdk.LoadDQLInput{
    DQL: string(source),
})
```

ZIP, TAR and TAR.GZ archives can contain multiple root `.dql` files and nested
dependencies. Select the component's entry document when multiple roots exist:

```go
result, err := client.Versions().LoadArchive(ctx, reportID, sdk.LoadArchiveInput{
    Archive: archiveBytes,
    Format: "zip", // also tar, tar.gz, tgz
    EntryDQL: "reader.dql",
})
```

All regular archive files are preserved as version resources with their relative
paths. For example, `${embed:sql/query.dql}` resolves the stored
`sql/query.dql` dependency through Datly's resource store. Root DQL documents
other than the selected entry are resources; they are not automatically
published as separate components. `Entries` returns all available root DQL
documents; call `LoadArchive` with each desired report/entry pair to import
multiple components.

The draft source, resources and current-draft pointer are stored atomically.
The result remains `pending`; call `Versions().Validate` before publication.
Compilation and linked Go-type availability are determined by the Datly host.

The HTTP SDK operations are `versions.load_dql` and `versions.load_archive`,
each accepting `{ "reportId": "...", "input": { ... } }`. JSON encodes archive
bytes as base64. Limits are 16 MiB compressed, 64 MiB expanded and 1024 archive
entries. Absolute/traversal paths, duplicate paths, links and special files are
rejected. Extraction is in memory and never writes archive paths to disk.

For inspection without persistence, use `sdk.ReadDQL(io.Reader)` or
`sdk.ReadDQLArchive(io.Reader, format)`.
