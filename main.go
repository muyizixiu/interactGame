package main

import (
	"io"
	"net/http"
	"os"
	"strings"
)
import "fmt"

import "ws"

func main() {
	go func() {
		println("server listen at 80 port")
		err := http.ListenAndServe(":80", httpHandler)
		fmt.Println(err.Error())
	}()
	println("server listen at 81 port")
	err := http.ListenAndServe(":81", ws.WsHandler)
	fmt.Println(err.Error())
}

// http handler
type Handler struct {
	Router map[string]string
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	println(r.URL.Path)
	h.Route(r.URL.Path, w, r)
}

func (h Handler) Route(path string, w http.ResponseWriter, r *http.Request) {
	path = strings.Trim(path, "/")
	if str, ok := h.Router[path]; ok {
		path = str
	}
	f, err := os.Open(path)
	if err != nil {
		println(err.Error())
		w.Write([]byte(err.Error()))
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

var httpHandler Handler

func init() {
	httpHandler.Router = map[string]string{
		"chess": "chess.html",
	}
}
