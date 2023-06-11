package main

import (
	"context"
	"crudd/shutdownlib"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
)

type data struct {
	Title  string
	Output string
}

var (
	port = flag.String("port", ":4901", "Server port")

	indexTemplate   = template.Must(template.ParseFiles("templates/index.html"))
	commandTemplate = template.Must(template.ParseFiles("templates/command.html"))

	topBin  = flag.String("top_bin", "/usr/bin/top", "Location of the top binary")
	topArgs = flag.String("top_args", "-bn1 -w256", "Args for the top binary")

	freeBin  = flag.String("free_bin", "/usr/bin/free", "Location of the free binary")
	freeArgs = flag.String("free_args", "-hw", "Args for the free binary")

	dfBin  = flag.String("df_bin", "/bin/df", "Location of the df binary")
	dfArgs = flag.String("df_args", "-h", "Args for the df binary")
)

func main() {
	flag.Parse()

	log.Printf("CRUDD is starting up")

	ctx := context.Background()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/df", dfHandler)
	http.HandleFunc("/free", freeHandler)
	http.HandleFunc("/top", topHandler)

	httpServer := &http.Server{
		Addr:    *port,
		Handler: logging()(http.DefaultServeMux),
	}

	shutdownlib.AddShutdownHandler(func() error {
		log.Println("CRUDD is shutting down")
		return httpServer.Shutdown(ctx)
	})

	log.Println("CRUDD is ready to handle requests at", *port)
	if err := httpServer.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", *port, err)
		}
	}

	shutdownlib.WaitForShutdown()
	log.Println("CRUDD has been shut down")
}

func logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				log.Println(r.Method, r.URL.Path, r.RemoteAddr)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := indexTemplate.Execute(w, nil); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func dfHandler(w http.ResponseWriter, r *http.Request) {
	writeCommandOutput(w, runCommand(*dfBin, *dfArgs))
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	writeCommandOutput(w, runCommand(*topBin, *topArgs))
}

func freeHandler(w http.ResponseWriter, r *http.Request) {
	writeCommandOutput(w, runCommand(*freeBin, *freeArgs))
}

func writeCommandOutput(w http.ResponseWriter, d *data) {
	if err := commandTemplate.Execute(w, d); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func runCommand(bin, args string) *data {
	cmd := exec.Command(bin, strings.Split(args, " ")...)

	log.Printf("Executing cmd: %s", cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return &data{
			Title:  fmt.Sprintf("%s %s", bin, args),
			Output: fmt.Sprintf("failed to run %s: %s", bin, err),
		}
	}

	return &data{
		Title:  fmt.Sprintf("%s %s", bin, args),
		Output: string(out),
	}
}
