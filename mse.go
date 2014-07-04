package main

import (
	// "bufio"
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"io"
	// "io/ioutil"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	// "time"
)

var source = NewSource()

type VideoSource struct {
	size   int
	hashes chan []byte
	buffer bytes.Buffer
}

func NewSource() *VideoSource {
	return &VideoSource{CHUNKSIZE, make(chan []byte), bytes.Buffer{}}
}

func (source *VideoSource) Write(p []byte) (int, error) {
	n, err := source.buffer.Write(p)
	if source.buffer.Len() >= source.size {
		h := sha1.New()
		h.Sum(source.buffer.Bytes())
		source.hashes <- h.Sum(nil)
	}
	return n, err
}

func (source *VideoSource) Read(p []byte) (int, error) {
	hash := <-source.hashes
	log.Println("buffer length:", source.buffer.Len())
	n, err := source.buffer.Read(p)
	h := sha1.New()
	h.Sum(p)
	log.Println(hex.EncodeToString(hash), hex.EncodeToString(h.Sum(nil)))
	log.Println(len(p), n, err)
	return n, err
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	CHUNKS    = 15
	CHUNKSIZE = 200 * 1024
)

// https://github.com/go-av/rtmp

// ffmpeg -i test.webm -f webm -vcodec vp8 -g 10 t.webm
// var command = fmt.Sprintf("ffmpeg -i - -c:v libvpx -b:v %dk -c:a libvorbis -f webm -threads %d -g 1 -", 500, 4)
var command = fmt.Sprintf("ffmpeg -i \"http://video14.fra01.hls.twitch.tv/hls36/appspy_10136390480_114031571/chunked/index-live.m3u8?token=id=4092849198343099829,bid=10136390480,exp=1404579263,node=video14-1.fra01.hls.justin.tv,nname=video14.fra01,fmt=chunked&sig=475815aa145bb0db354b0e00372477277692caf8\" -c:v libvpx -b:v %dk -c:a libvorbis -f webm -threads %d -g 1 -", 500, 4)

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
	log.Println("chunk size:", chunkSize)
	log.Println("[realtime]", "connected")

	cmd := exec.Command("/bin/bash", "-c", command)
	// cmd.Stdin = f
	cmd.Stdout = source
	cmd.Stderr = os.Stderr

	go cmd.Start()

	defer func() {
		conn.Close()
		log.Println("[realtime] closed")
	}()

	for {
		log.Println("reading")
		chunk := make([]byte, CHUNKSIZE)
		n, err := source.Read(chunk)
		log.Println("read", n)
		conn.WriteMessage(websocket.BinaryMessage, chunk[:n])
		log.Println("sent", n)
		if err == io.EOF {
			log.Println(err)
			break
		}
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
