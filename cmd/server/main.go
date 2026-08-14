package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wlnyx/fenbi-helper-go/internal/web"
)

func main() {
	port := flag.String("port", "3000", "监听端口")
	dataDir := flag.String("data", "", "数据目录（默认: 工作目录 .data）")
	flag.Parse()

	if *dataDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		*dataDir = filepath.Join(cwd, ".data")
	}
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	srv, err := web.NewServer(*dataDir)
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + *port
	log.Printf("粉笔复盘工作台运行于 http://localhost:%s （数据目录: %s）", *port, *dataDir)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}
