# `headproxy`

A Go template function that outputs the `<head>` element's content for a given domain.

### Usage

```go
package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/bentranter/headproxy"
)

const htmltpl = `<!DOCTYPE html>
<html>
  <head>
    {{ headproxy "bentranter.io" }}
  </head>
  <body>
    <h1>Hello, world!</h1>
  </body>
</html>
`

func main() {
	tpl := template.New("home")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error
		tpl, err = tpl.Funcs(headproxy.Map(w, r)).Parse(htmltpl)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		tpl.ExecuteTemplate(w, "home", nil)
	})

	log.Println("Starting on localhost:3000")
	log.Fatalln(http.ListenAndServe(":3000", nil))
}
```
