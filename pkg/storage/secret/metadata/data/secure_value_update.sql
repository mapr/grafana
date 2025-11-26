UPDATE
  {{ .Ident "secret_secure_value" }}
SET
  {{ .Ident "annotations" }} = {{ .Arg .Row.Annotations }},
  {{ .Ident "labels" }} = {{ .Arg .Row.Labels }},
  {{ .Ident "updated" }} = {{ .Arg .Row.Updated }},
  {{ .Ident "updated_by" }} = {{ .Arg .Row.UpdatedBy }},
  {{ .Ident "active" }} = {{ .Arg .Row.Active }},
  {{ .Ident "description" }} = {{ .Arg .Row.Description }},
  {{ if .Row.Decrypters.Valid }}
    {{ .Ident "decrypters" }} = {{ .Arg .Row.Decrypters.String }},
  {{ end }}
  {{ if .Row.Ref.Valid }}
    {{ .Ident "ref" }} = {{ .Arg .Row.Ref.String }},
  {{ end }}
  {{ .Ident "external_id" }} = {{ .Arg .Row.ExternalID }}
WHERE
  {{ .Ident "namespace" }} = {{ .Arg .Row.Namespace }} AND
  {{ .Ident "name" }} = {{ .Arg .Row.Name }} AND
  {{ .Ident "version" }} = {{ .Arg .Row.Version }}
;
