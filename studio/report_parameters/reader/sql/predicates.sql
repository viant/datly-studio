SELECT predicates."report_id", predicates."version_no", predicates."parameter_id", predicates."predicate_index", predicates."predicate_group", predicates."name", predicates."args_json", predicates."apply_when_absent" FROM (
    SELECT p.report_id, p.version_no, p.parameter_id, p.predicate_index,
       p.predicate_group, p.name, p.args_json, p.apply_when_absent
FROM report_predicates p

) predicates