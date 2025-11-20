DELETE FROM
  {{ .Ident "secret_data_key" }}
WHERE {{ .Ident "active" }} = false
;
