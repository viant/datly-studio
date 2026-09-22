SELECT predicate."report_id", predicate."version_no", predicate."parameter_id", predicate."predicate_index", predicate."predicate_group", predicate."name", predicate."args_json", predicate."apply_when_absent", predicate."should_delete" FROM (
    SELECT p.*, '' AS should_delete
FROM report_predicates p

) predicate