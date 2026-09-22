SELECT r."args_json", r."report_id", r."version_no", r."parameter_id", r."predicate_index", r."name", r."predicate_group", r."apply_when_absent" FROM (SELECT predicate."report_id", predicate."version_no", predicate."parameter_id", predicate."predicate_index", predicate."predicate_group", predicate."name", predicate."args_json", predicate."apply_when_absent", predicate."should_delete" FROM (
    SELECT p.*, '' AS should_delete
FROM report_predicates p

) predicate) r WHERE $criteria.CompositeIn("r", $Unsafe.ProjectCurrentPredicateParentKeys($CurrentParameter))