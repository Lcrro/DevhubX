package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"devhub/internal/server"
	"devhub/web"
)

func main() {
	address := flag.String("addr", "127.0.0.1:4780", "Loopback listen address")
	config, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	data := flag.String("data", filepath.Join(config, "DevHub"), "Private local data directory")
	webDir := flag.String("web", "", "Optional frontend directory (development)")
	flag.Parse()
	port, err := server.ListenAddress(*address)
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	abs, err := filepath.Abs(*data)
	if err != nil {
		log.Fatal(err)
	}
	app, err := server.New(abs, port)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	assets, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		log.Fatal(err)
	}
	if *webDir != "" {
		assets = os.DirFS(*webDir)
	}
	httpServer := &http.Server{Addr: *address, Handler: app.Handler(assets), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	app.Start()
	fmt.Printf("DevHub ready: http://%s\nData: %s\n", *address, abs)
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Print(err)
	}
}
