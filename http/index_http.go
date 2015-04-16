package http

import (
	"github.com/falcon/task/index"
	"net/http"
)

func configIndexHttpRoutes() {
	http.HandleFunc("/index/delete", func(w http.ResponseWriter, r *http.Request) {
		index.DeleteIndex()
		RenderDataJson(w, "done")
	})
}
