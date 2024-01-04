//go:build generate

package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"slices"
	"text/template"
	"time"

	"github.com/AdguardTeam/AdGuardDNS/internal/agdhttp"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
)

func main() {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, fakeECSBlocklistURL, nil)
	check(err)

	req.Header.Add(httphdr.UserAgent, agdhttp.UserAgent())

	resp, err := c.Do(req)
	check(err)
	defer log.OnCloserError(resp.Body, log.ERROR)

	out, err := os.OpenFile("./ecsblocklist.go", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o664)
	check(err)
	defer log.OnCloserError(out, log.ERROR)

	contents, err := io.ReadAll(resp.Body)
	check(err)

	lines := bytes.Split(contents, []byte("\n"))
	lines = lines[:len(lines)-1]

	slices.SortStableFunc(lines, bytes.Compare)

	tmpl, err := template.New("main").Parse(tmplStr)
	check(err)

	err = tmpl.Execute(out, lines)
	check(err)
}

// fakeECSBlocklistURL is the default URL from where to get ECS fake domains.
const fakeECSBlocklistURL = `https://filters.adtidy.org/dns/fake-ecs-blacklist`

// tmplStr is the template of the generated Go code.
const tmplStr = `// Code generated by go run ./ecsblocklist_generate.go; DO NOT EDIT.

package ecscache

// FakeECSFQDNs contains all domains that indicate ECS support, but in fact
// don't have one.
var FakeECSFQDNs = map[string]struct{}{
{{- range $_, $h := . }}
	{{ printf "\"%s.\": {}," $h }}
{{- end }}
}
`

// check is a simple error checker.
func check(err error) {
	if err != nil {
		panic(err)
	}
}
