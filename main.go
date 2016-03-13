package main

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
	println("hel")
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
		f, err := os.Open("./chess.html")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		io.Copy(w, f)
	})
	go func() {
		err := http.ListenAndServe(":80", nil)
		println("ehll")
		fmt.Println(err.Error())
	}()
	err := http.ListenAndServe(":81", ws.WsHandler)
	fmt.Println(err.Error())
}
