package main /*package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
)
import "fmt"

import "ws"

func main() {
	http.HandleFunc("/gate", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		room := r.Form.Get("room")
		var roomId int
		if i, err := strconv.ParseInt(room, 10, 64); err == nil {
			if i != 0 {
				roomId = int(i)
			}
		}
		name := r.Form.Get("name")
		a := ws.Key{RoomId: roomId, Name: name}
		byt, _ := json.Marshal(a)
		w.Write(byt)
	})
	http.HandleFunc("/chess", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var f *os.File
		var err error
		if r.Form.Get("dev") == "mobile" {
			f, err = os.Open("chess.mobile.html")
		} else {
			f, err = os.Open("chess.html")
		}
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.HandleFunc("/flappy", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var f *os.File
		var err error
		if r.Form.Get("dev") == "mobile" {
			f, err = os.Open("chess.mobile.html")
		} else {
			f, err = os.Open("muyizixiu/flappy-block.html")
		}
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.HandleFunc("/flat.png", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("flat.png")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.HandleFunc("/web.png", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("web.png")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.HandleFunc("/loading.gif", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("loading.gif")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	go func() {
		println("server listen at 80 port")
		err := http.ListenAndServe(":80", nil)
		fmt.Println(err.Error())
	}()
	println("server listen at 81 port")
	err := http.ListenAndServe(":81", ws.WsHandler)
	fmt.Println(err.Error())
}
*/
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
		"chess":         "chess.html",
		"flappy":        "muyizixiu/flappy-block.html",
		"flappy.mobile": "muyizixiu/flappy-mobile.html",
	}
}
