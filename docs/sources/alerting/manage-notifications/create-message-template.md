---
aliases:
  - /docs/grafana/latest/alerting/contact-points/message-templating/
  - /docs/grafana/latest/alerting/contact-points/message-templating/create-message-template/
  - /docs/grafana/latest/alerting/message-templating/
  - /docs/grafana/latest/alerting/unified-alerting/message-templating/
  - /docs/grafana/latest/alerting/contact-points/message-templating/delete-message-template/
  - /docs/grafana/latest/alerting/contact-points/message-templating/edit-message-template/
  - /docs/grafana/latest/alerting/manage-notifications/create-message-template/
  - /docs/grafana/latest/alerting/contact-points/message-templating/
  - /docs/grafana/latest/alerting/contact-points/message-templating/example-template/
  - /docs/grafana/latest/alerting/message-templating/
  - /docs/grafana/latest/alerting/unified-alerting/message-templating/
  - /docs/grafana/latest/alerting/fundamentals/contact-points/example-template/
  - /docs/grafana/latest/alerting/contact-points/message-templating/template-data/
  - /docs/grafana/latest/alerting/message-templating/template-data/
  - /docs/grafana/latest/alerting/unified-alerting/message-templating/template-data/
  - /docs/grafana/latest/alerting/fundamentals/contact-points/template-data/
keywords:
  - grafana
  - alerting
  - guide
  - contact point
  - templating
title: Templating notifications
weight: 200
---

# An introduction to templating notifications

In Grafana it is possible to customize notifications with templates. All notifications templates are written Go templates, also known as [text/template](https://pkg.go.dev/text/template). Templates can be used to change the subject, message, and the formatting of notifications such as bold and italic text and line breaks.

Customizing notifications with templates can be done via either writing all template code in the contact point or creating a separate message template and referencing it from the contact point. For example, if the template is simple, and is intended to be used in a single contact point, then it might be better to write all the template code in the contact point. However, if the template is complex, or is intended to be shared between a number of different contact points, then it would be better to create a message template and reference it from the contact point.

## Writing all template code in the contact point

## Creating a message template and referencing it from a contact point

To create a message template, complete the following steps.

1. In the Grafana menu, click the **Alerting** (bell) icon to open the Alerting page listing existing alerts.
2. In the Alerting page, click **Contact points** to open the page listing existing contact points.
3. From Alertmanager drop-down, select an external Alertmanager to create and manage templates for the external data source. Otherwise, keep the default option of Grafana.
   {{< figure max-width="250px" src="/static/img/docs/alerting/unified/contact-points-select-am-8-0.gif" caption="Select Alertmanager" >}}
4. Click **Add template**.
5. In **Name**, add a descriptive name.
6. In **Content**, add the content of the template.
7. Click **Save template** button at the bottom of the page.
   <img  src="/static/img/docs/alerting/unified/templates-create-8-0.png" width="600px">

The `define` tag in the Content section assigns the template name. This tag is optional, and when omitted, the template name is derived from the **Name** field. When both are specified, it is a best practice to ensure that they are the same.

### Edit a message template:

To edit a message template, complete the following steps.

1. In the Alerting page, click **Contact points** to open the page listing existing contact points.
1. In the Template table, find the template you want to edit, then click the **Edit** (pen icon).
1. Make your changes, then click **Save template**.

### Delete a message template:

To delete a message template, complete the following steps.

1. In the Alerting page, click **Contact points** to open the page listing existing contact points.
1. In the Template table, find the template you want to delete, then click the **Delete** (trash icon).
1. In the confirmation dialog, click **Yes, delete** to delete the template.

Use caution when deleting a template since Grafana does not prevent you from deleting templates that are in use.

### Create a custom template

Here's an example of how to use a custom template. You can also use the default template included in the setup.

Step 1: Configure a template to render a single alert.

## Writing templates in text/template

Notification templates are written in Go templates, also known as [text/template](https://pkg.go.dev/text/template), and is a little different from other popular templating languages such as Jinja. In text/template, template code starts with `{{` and ends with `}}` irrespective of whether the template code prints a variable or executes control structures such as if statements. This is different from other templating languages such as Jinja where printing a variable uses `{{` and `}}` and control structures use `{%` and `%}`.

We will next look at some of the basics in text/template before looking a number of more complex examples.

### Printing variables

Unlike other templating languages which uses variables, in text/template there is a special cursor called dot (written as `.`). You can think of this cursor as a variable whose value changes depending where in the template it is used. For example, at the start of all notification templates the cursor, called dot, contains a number of fields including `Alerts`, `Status`, `GroupLabels`, `CommonLabels`, `CommonAnnotations` and `ExternalURL`. For example, to print all alerts we could write the following template code:

```
{{ .Alerts }}
```

### Iterating over alerts

What if instead of printing all labels, annotations, and metadata for all alerts we just want to print the labels for each alert? We cannot just write a template with `{{ .Labels }}` because `Labels` does not exist at the start of our notification template. Instead, we must use a `range` to iterate the alerts in `.Alerts`:

```
{{ range .Alerts }}
{{ .Labels }}
{{ end }}
```

You might have noticed that inside the `range` we can write `{{ .Labels }}` to print the labels of each alert. This works because `range .Alerts` changes the cursor to refer to the current alert in the list of alerts. When the range is finished the cursor is reset to the value it had before the start of the range:

```
{{ range .Alerts }}
{{ .Labels }}
{{ end }}
{{/* does not work, .Labels does not exist here */}}
{{ .Labels }}
{{/* works, cursor was reset */}}
{{ .Status }}
```

### Iterating over annotations and labels

Now that have a good understanding of `range` and how the cursor works let's write a template to print the labels of each alert in the format `The name of the label is $name, and the value is $value`, where `$name` and `$value` contain the name and value of each label.

Like in the previous example, we need to use a range to iterate over the alerts in `.Alerts` and change the cursor to refer to the current alert in the list. However, we then need to use a second range on the sorted labels so the cursor is updated once more to refer to each individual label pair in `.Labels.SortedPairs`. Here we can use `.Name` and `.Value` to print the name and value of each label:

```
{{ range .Alerts }}
{{ range .Labels.SortedPairs }}
The name of the label is {{ .Name }}, and the value is {{ .Value }}
{{ end }}
{{ end }}
```

It is important to understand that in the second range it is not possible to use `.Labels` as the cursor is now referring to the current label pair in the current alert:

```
{{ range .Alerts }}
{{ range .Labels.SortedPairs }}
The name of the label is {{ .Name }}, and the value is {{ .Value }}
{{/* does not work because in the second range . is a label not an alert */}}
{{ .Labels.SortedPairs }}
{{ end }}
{{ end }}
```

It is possible to get around this using variables, which we will look at next.

### Variables

While text/template has a cursor, it is still possible to define variables. However, unlike other templating languages such as Jinja where variables refer to data passed into the template, variables in text/template must be created within the template.

The following example creates a variable called `variable` with the current value of the cursor:

```
{{ $variable := . }}
```

We can use this to create a variable called `$alert` to fix the issue we had in the previous example:

```
{{ range .Alerts }}
{{ $alert := . }}
{{ range .Labels.SortedPairs }}
The name of the label is {{ .Name }}, and the value is {{ .Value }}
{{/* works because we created a variable called $alert */}}
{{ $alert.SortedPairs }}
{{ end }}
{{ end }}
```

### If statements

If statements are supported in text/template too. For example to print `There are no alerts` if there are no alerts in `.Alerts` we could write the following template code:

```
{{ if .Alerts }}
{{ range .Alerts }}
{{ .Labels }}
{{ end }}
{{ else }}
There are no alerts
{{ end }}
```

### Indentation

It is possible to use indentation to make templates more readable:

```
{{ if .Alerts }}
  {{ range .Alerts }}
    {{ .Labels }}
  {{ end }}
{{ else }}
  There are no alerts
{{ end }}
```

However, such intention is then included in the printed text. Next we will see how to remove it.

### Removing spaces and line breaks

Suppose we have the following template:

```
{{ range .Alerts }}
{{ range .Labels.SortedPairs }}
{{ .Name }} = {{ .Value }}
{{ end }}
{{ end }}
```

The output of this template might be:

```
alertname = High CPU usage
instance = server1
```

But what if we want:

```
alertname = High CPU usage instance = server1
```

In text/template we can use `{{-` and `-}}` to remove leading and trailing spaces and line breaks:

```
{{ range .Alerts -}}
{{ range .Labels.SortedPairs -}}
{{ .Name }} = {{ .Value }}
{{- end }}
{{- end }}
```

We can use `{{-` and `-}}` to remove indentation too:

```
{{ if .Alerts -}}
  {{ range .Alerts -}}
    {{ .Labels }}
  {{ end -}}
{{ else -}}
  There are no alerts
{{- end }}
```

### Comments

It is possible to add comments with `{{/*` and `*/}}`:

```
{{/* This is a comment */}}
```

### Defining a template

When writing complex template code, or template code intended to be shared across a number of different contact points, it is recommended to define it within a template. Here we are defining a template called `print_labels`:

```
{{ define "print_labels" }}
{{ range .Alerts }}
{{ range .Labels.SortedPairs }}
The name of the label is {{ .Name }}, and the value is {{ .Value }}
{{ end }}
{{ end }}
{{ end }}
```

When defining a template it is important to make sure the name is unique. It is not recommend definining templates with the same name as default templates such as `__subject`, `__text_values_list`, `__text_alert_list`, `default.title` and `default.message`.

### Executing a template

To execute a template use `template`, passing as arguments the name of the template in double quotes, and the cursor that should be passed into the template:

```
{{ template "print_labels" . }}
```

### Templates and cursors

It is possible to pass cursors into templates other than the dot cursor. For example, we can change the `print_labels` template to iterate over `.` instead of `.Alerts`:

```
{{ define "print_labels" }}
{{ range . }}
{{ range .Labels.SortedPairs }}
The name of the label is {{ .Name }}, and the value is {{ .Value }}
{{ end }}
{{ end }}
{{ end }}
```

When executing the template, instead of passing it the original dot cursor we would pass the list of alerts which as we know from previous examples is `.Alerts`:

```
{{ template "print_labels" .Alerts }}
```

The advantage of this is that we can either pass all alerts to the template, or just the firing alerts, without having the change the template:

```
{{ template "print_labels" .Alerts.Firing }}
```

To avoid comments from adding line breaks use:

```
{{- /* This is a comment with no leading or trailing line breaks */ -}}
```

## Example templates

Now that we have looked at some of the basics of text/template let's look at a some more complex examples:

### Print the labels, annotations, SilenceURL and DashboardURL of all alerts

```
{{ define "custom.print_alert" }}
  [{{.Status}}] {{ .Labels.alertname }}

  Labels:
  {{ range .Labels.SortedPairs }}
    {{ .Name }}: {{ .Value }}
  {{ end }}

  {{ if gt (len .Annotations) 0 }}
  Annotations:
  {{ range .Annotations.SortedPairs }}
    {{ .Name }}: {{ .Value }}
  {{ end }}
  {{ end }}

  {{ if gt (len .SilenceURL ) 0 }}
    Silence alert: {{ .SilenceURL }}
  {{ end }}
  {{ if gt (len .DashboardURL ) 0 }}
    Go to dashboard: {{ .DashboardURL }}
  {{ end }}
{{ end }}
```

```
{{ define "custom.message" }}
  {{ if gt (len .Alerts.Firing) 0 }}
    {{ len .Alerts.Firing }} firing:
    {{ range .Alerts.Firing }} {{ template "custom.print_alert" .}} {{ end }}
  {{ end }}
  {{ if gt (len .Alerts.Resolved) 0 }}
    {{ len .Alerts.Resolved }} resolved:
    {{ range .Alerts.Resolved }} {{ template "custom.print_alert" .}} {{ end }}
  {{ end }}
{{ end }}
```

```
{{ template "mymessage" . }}
```

## Reference

### ExtendedData

| Name              | Dot notation         | Kind        | Notes                                                                                                                |
| ----------------- | -------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------- |
| Receiver          | `.Receiver`          | string      | Name of the contact point that the notification is being sent to.                                                    |
| Status            | `.Status`            | string      | `firing` if at least one alert is firing, otherwise `resolved`.                                                      |
| Alerts            | `.Alerts`            | Alert       | List of alert objects that are included in this notification (see below).                                            |
| GroupLabels       | `.GroupLabels`       | Named Pairs | Labels these alerts were grouped by.                                                                                 |
| CommonLabels      | `.CommonLabels`      | Named Pairs | Labels common to all the alerts included in this notification.                                                       |
| CommonAnnotations | `.CommonAnnotations` | Named Pairs | Annotations common to all the alerts included in this notification.                                                  |
| ExternalURL       | `.ExternalURL`       | string      | Back link to the Grafana that sent the notification. If using external Alertmanager, back link to this Alertmanager. |

The `Alerts` type exposes functions for filtering alerts:

- `Alerts.Firing` returns a list of firing alerts.
- `Alerts.Resolved` returns a list of resolved alerts.

### Alert

| Name         | Dot notation    | Kind                                 | Notes                                                                                                                                          |
| ------------ | --------------- | ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Status       | `.Status`       | string                               | `firing` or `resolved`.                                                                                                                        |
| Labels       | `.Labels`       | Named Pairs                          | A set of labels attached to the alert.                                                                                                         |
| Annotations  | `.Annotations`  | Named Pairs                          | A set of annotations attached to the alert.                                                                                                    |
| StartsAt     | `.StartsAt`     | [Time](https://pkg.go.dev/time#Time) | Time the alert started firing.                                                                                                                 |
| EndsAt       | `.EndsAt`       | [Time](https://pkg.go.dev/time#Time) | Only set if the end time of an alert is known. Otherwise set to a configurable timeout period from the time since the last alert was received. |
| GeneratorURL | `.GeneratorURL` | string                               | A back link to Grafana or external Alertmanager.                                                                                               |
| SilenceURL   | `.SilenceURL`   | string                               | Link to grafana silence for with labels for this alert pre-filled. Only for Grafana managed alerts.                                            |
| DashboardURL | `.DashboardURL` | string                               | Link to grafana dashboard, if alert rule belongs to one. Only for Grafana managed alerts.                                                      |
| PanelURL     | `.PanelURL`     | string                               | Link to grafana dashboard panel, if alert rule belongs to one. Only for Grafana managed alerts.                                                |
| Fingerprint  | `.Fingerprint`  | string                               | Fingerprint that can be used to identify the alert.                                                                                            |
| ValueString  | `.ValueString`  | string                               | A string that contains the labels and value of each reduced expression in the alert.                                                           |

### GroupLabels, CommonLabels and CommonAnnotations

`KeyValue` is a set of key/value string pairs that represent labels and annotations.

Here is an example containing two annotations:

```json
{
  "summary": "alert summary",
  "description": "alert description"
}
```

In addition to direct access of data (labels and annotations) stored as KeyValue, there are also methods for sorting, removing and transforming.

| Name        | Arguments | Returns                                 | Notes                                                       |
| ----------- | --------- | --------------------------------------- | ----------------------------------------------------------- |
| SortedPairs |           | Sorted list of key & value string pairs |
| Remove      | []string  | KeyValue                                | Returns a copy of the Key/Value map without the given keys. |
| Names       |           | []string                                | List of label names                                         |
| Values      |           | []string                                | List of label values                                        |
