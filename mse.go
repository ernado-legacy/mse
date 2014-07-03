package main

import (
	// "bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"io"
	// "io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	CHUNKS = 15
)

// https://github.com/go-av/rtmp

// ffmpeg -i test.webm -f webm -vcodec vp8 -g 10 t.webm
var command = fmt.Sprintf("ffmpeg -i - -c:v libvpx -b:v %dk -c:a libvorbis -f webm -threads %d -g 1 -", 500, 4)

func handleConnection(conn net.Conn) {
	for {
		log.Println("tcp reading")
		chunk := make([]byte, 32000)
		n, err := conn.Read(chunk)
		log.Println(string(chunk[:n]))
		log.Println("tcp read", n)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func Listen() {
	log.Println("listening")
	ln, err := net.Listen("tcp", ":1935")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnection(conn)
	}
}

func Realtime(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	f, _ := os.Open("t.webm")
	stat, _ := f.Stat()
	chunkSize := stat.Size() / CHUNKS
	reader, writer := io.Pipe()
	log.Println("chunk size:", chunkSize)
	log.Println("[realtime]", "connected")

	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Stdin = f
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	go cmd.Start()

	defer func() {
		conn.Close()
		log.Println("[realtime] closed")
	}()

	for {
		log.Println("reading")
		chunk := make([]byte, chunkSize)
		n, err := reader.Read(chunk)
		log.Println("read", n)
		conn.WriteMessage(websocket.BinaryMessage, chunk[:n])
		log.Println("sent", n)
		if err == io.EOF {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	router := httprouter.New()
	router.ServeFiles("/static/*filepath", http.Dir("."))
	router.GET("/realtime", Realtime)
	log.Println("listening on :5555")
	go Listen()
	log.Fatal(http.ListenAndServe(":5555", router))
}
