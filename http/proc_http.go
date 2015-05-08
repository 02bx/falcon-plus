package http

import (
	"github.com/open-falcon/graph/g"
	"github.com/open-falcon/graph/proc"
	"net/http"
	"strings"
)

func configProcRoutes() {
	// TOP
	http.HandleFunc("/statistics/all", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, proc.GetAll())
	})

	http.HandleFunc("/statistics/config", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, g.Config())
	})

	// trace
	http.HandleFunc("/trace/", func(w http.ResponseWriter, r *http.Request) {
		urlParam := r.URL.Path[len("/trace/"):]
		args := strings.Split(urlParam, "/")

		argsLen := len(args)
		endpoint := args[0]
		metric := args[1]
		tags := make(map[string]string)
		if argsLen > 2 {
			tagVals := strings.Split(args[2], ",")
			for _, tag := range tagVals {
				tagPairs := strings.Split(tag, "=")
				if len(tagPairs) == 2 {
					tags[tagPairs[0]] = tagPairs[1]
				}
			}
		}
		proc.RecvDataTrace.SetTraceConfig(endpoint, metric, tags)
		RenderDataJson(w, proc.RecvDataTrace.FilterAll())
	})

}
