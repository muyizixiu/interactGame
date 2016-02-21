package main

import "net/http"
import "fmt"
import "os"
import "io"
import "ws"

func main() {
	println("hel")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("html/ws.html")
		if err != nil {
			fmt.Println(err.Error())
		}
		io.Copy(w, f)
	})
	http.HandleFunc("/mv", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("html/mv.html")
		if err != nil {
			fmt.Println(err.Error())
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
