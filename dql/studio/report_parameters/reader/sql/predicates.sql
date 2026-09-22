SELECT p.report_id, p.version_no, p.parameter_id, p.predicate_index,
       p.predicate_group, p.name, p.args_json, p.apply_when_absent
FROM report_predicates p
