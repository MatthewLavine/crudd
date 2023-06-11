package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/template"
	"time"
)

const (
	indexTemplate = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>CRUDD</title>
		<style>
			* {
				background-color: #121212;
				color: #ffffff;
			}
			body {
				padding: 10px;
			}
		</style>
	</head>
	<body>
		<h1>Continuously Running Userland Diagnostics Daemon</h1>
		<h2><a href="/df">/df</a></h2>
		<h2><a href="/free">/free</a></h2>
		<h2><a href="/top">/top</a></h2>
	</body>
</html>`

	commandTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>{{.Title}}</title>
	<style>
		* {
			background-color: #121212;
			color: #ffffff;
		}
	</style>
</head>
<body>
	<pre>{{.Output}}</pre>
</body>
</html>`
)

type data struct {
	Title  string
	Output string
}

var (
	port = flag.String("port", ":4901", "Server port")

	topBin  = flag.String("top_bin", "/usr/bin/top", "Location of the top binary")
	topArgs = flag.String("top_args", "-bn1 -w256", "Args for the top binary")

	freeBin  = flag.String("free_bin", "/usr/bin/free", "Location of the free binary")
	freeArgs = flag.String("free_args", "-hw", "Args for the free binary")

	dfBin  = flag.String("df_bin", "/bin/df", "Location of the df binary")
	dfArgs = flag.String("df_args", "-h", "Args for the df binary")
)

func main() {
	flag.Parse()

	log.Printf("CRUDD starting up")

	http.HandleFunc("/", indexHandler)

	http.HandleFunc("/df", dfHandler)

	http.HandleFunc("/free", freeHandler)

	http.HandleFunc("/top", topHandler)

	server := &http.Server{
		Addr: *port,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Println("CRUDD is shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Println("CRUDD is ready to handle requests at", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", *port, err)
	}

	<-done
	log.Println("CRUDD has been shut down")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		fmt.Fprintf(w, "failed to parse template: %v", err)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
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
	tmpl, err := template.New("command").Parse(commandTemplate)
	if err != nil {
		fmt.Fprintf(w, "failed to parse template: %v", err)
		return
	}
	if err := tmpl.Execute(w, d); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func runCommand(bin, args string) *data {
	cmd := exec.Command(bin, strings.Split(args, " ")...)

	log.Printf("Running cmd: %s", cmd)

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
