CREATE TABLE discovery_items (id INTEGER PRIMARY KEY, dataset_marker TEXT);
INSERT INTO discovery_items VALUES (1, 'lookup');

CREATE TABLE discovery_extra (id INTEGER PRIMARY KEY, extra_marker TEXT);
INSERT INTO discovery_extra VALUES (1, 'extra');

WITH RECURSIVE numbers(id) AS (SELECT 2 UNION ALL SELECT id+1 FROM numbers WHERE id<61)
INSERT INTO discovery_extra(id,extra_marker) SELECT id,'extra' FROM numbers;
