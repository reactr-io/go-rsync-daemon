package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Executes at most, one rsync request at a time
func worker(src, dst, webhook string, ch chan int) {
	dirSeparator := string(os.PathSeparator)
	for {
		<-ch
		src := strings.TrimRight(src, dirSeparator) + dirSeparator

		cmd := exec.Command(
			"rsync",
			"-av",
			"--existing",
			src,
			dst,
		)

		log.Println("Syncing", src, "with", dst)

		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			log.Println(err)
		} else if len(webhook) > 0 {
			http.Get(webhook)
		}
	}
}

func main() {
	// Get parameters from command-line argument flags
	port := flag.String("listen", "localhost:18888", "the host/port which the http server should listen on")
	src := flag.String("src", "", "the source directory to sync")
	dst := flag.String("dst", "", "the destination directory to sync")
	webhook := flag.String("webhook", "", "a url endpoint you'd like to ping once rsync has completed")
	flag.Parse()

	log.SetOutput(os.Stdout)

	if len(*src) == 0 || len(*dst) == 0 || len(*port) == 0 {
		// If we don't have what we need, show usage
		flag.Usage()
	} else {
		// Otherwise, start the worker, and listen for HTTP requests
		queue := make(chan int)
		go worker(*src, *dst, *webhook, queue)

		http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Content-Type", "application/json")

			// Queue rsync job
			queue <- time.Now().Nanosecond()
			fmt.Fprintln(response, "{queued: true}")
		})
		http.ListenAndServe(*port, nil)
	}
}
