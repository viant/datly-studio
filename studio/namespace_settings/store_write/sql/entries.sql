SELECT * FROM (SELECT s.*, FALSE AS should_delete FROM namespace_settings s
) entries