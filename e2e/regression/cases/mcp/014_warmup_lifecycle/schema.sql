CREATE TABLE warmupcase_customers (id INTEGER PRIMARY KEY,tenant TEXT NOT NULL,name TEXT NOT NULL,active INTEGER NOT NULL);
INSERT INTO warmupcase_customers VALUES (10,'warmup','Alpha',1),(20,'warmup','Beta',0);
