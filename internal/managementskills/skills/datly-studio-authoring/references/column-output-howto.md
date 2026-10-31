# How to declare column output formatting

Reuse native Datly v1 column tags before adding SQL string conversions or a
consumer formatter. On an outer reader projection, a typed date can be declared:

```sql
SELECT readings.*,
       CAST(readings.day AS *time.Time),
       tag(readings.day, 'json:"day" format:"dateFormat=YYYY-MM-DD" groupable:"true"')
FROM (...) readings
```

This is an annotation fragment, not a complete runnable query. Keep the existing
source relation and required scope predicates. CAST declares the Go shape when
discovery needs it; do not add a SQL DATE_TRUNC to format the response.

Tagly translates `dateFormat=YYYY-MM-DD` into Go layout `2006-01-02`. The output
encoder emits a date-only string; it does not mutate the stored time, query grain
or date arithmetic. With no timezone override, the original value's calendar day
is retained. A raw time field without this declaration keeps its ordinary datetime
serialization. Null pointer values must remain null, not an epoch or empty date.

Native verification exists in Datly v1's
`report/TestDerivedCubePreservesSourceCivilDatePresentation` and Tagly's format
tests: tagged date, raw datetime and null output are checked independently.
Verify the deployed revision and the actual endpoint as well as source syntax.

## Gotchas

- Typed component invocation is not itself JSON encoding. A wrapper that extracts
  raw values and marshals them separately can bypass native output formatting.
- A derived cube and a composed result are different boundaries. Do not claim
  that a passing derived-cube test proves formatting after arbitrary composition.
- Preserve column tags/metadata through each boundary and use the native output
  plan. Do not hide a lost tag by truncating timestamps in a chart.
- If an explicit timezone conversion is intended, declare it and test a value
  near midnight; formatting alone is not a timezone conversion.
