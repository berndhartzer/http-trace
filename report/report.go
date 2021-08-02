package report

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/berndhartzer/http-trace/trace"
)

var outputTmpl = `> {{ .Request.Method }} {{ .Request.URL.Host }}{{ .Request.URL.Path }} {{ .Request.Proto }}
{{- range $key, $value := .Request.Header }}
> {{ $key }}: {{stringsJoin $value "" }}
{{- end }}
>
< {{ .Response.Status }}
{{- if not .Presentation.SuppressHeaders }}
{{- range $key, $value := .Response.Header }}
< {{ $key }}: {{stringsJoin $value "" }}
{{- end }}
{{- end }}
{{- if not .Presentation.SuppressBody }}
{{ .ResponseBody }}
{{- end }}

Trace
  Request
    Connection
      DNS Resolution:  {{ durationMillis .Timings.DNSDuration }}
      Connecting:      {{ durationMillis .Timings.ConnectionDialDuration }}
      TLS handshake:   {{ durationMillis .Timings.TLSDuration }}
    Connection total:  {{ durationMillis .Timings.TotalConnectionDuration }}

    Request write:     {{ durationMillis .Timings.RequestWriteDuration }}
    Response delay:    {{ durationMillis .Timings.ResponseDelayDuration }}
    Response read:     {{ durationMillis .Timings.ResponseReadDuration }}

  Request total:       {{ durationMillis .Timings.TotalRequestDuration }}
`

var tmplFuncs = template.FuncMap{
	"durationMillis": func(duration time.Duration) string {
		millisFloat := duration.Seconds() * 1000
		return fmt.Sprintf("%9.2fms", millisFloat)
	},
	"stringsJoin": strings.Join,
}

type Presentation struct {
	SuppressHeaders bool
	SuppressBody    bool
}

type reportData struct {
	Request      *http.Request
	Response     *http.Response
	ResponseBody string
	Timings      *trace.Timings
	Presentation *Presentation
}

type Report struct {
	data   *reportData
	output string
}

func New(req *http.Request, res *http.Response, body string, result *trace.Timings, pres *Presentation) *Report {
	data := &reportData{
		Request:      req,
		Response:     res,
		ResponseBody: body,
		Timings:      result,
		Presentation: pres,
	}

	return &Report{
		data: data,
	}
}

func (r *Report) Build() error {
	b := &bytes.Buffer{}

	tmpl := template.Must(template.New("output").Funcs(tmplFuncs).Parse(outputTmpl))
	err := tmpl.Execute(b, r.data)
	if err != nil {
		return fmt.Errorf("Error building report: %w", err)
	}

	r.output = b.String()
	return nil
}

func (r *Report) Print(w io.Writer) error {
	_, err := fmt.Fprint(w, r.output)
	if err != nil {
		return fmt.Errorf("Error writing output: %w", err)
	}

	return nil
}
