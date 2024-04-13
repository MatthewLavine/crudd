package main

import (
	"context"
	"crudd/commandlib"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"

	"github.com/MatthewLavine/gracefulshutdown"
)

const (
	indexTemplatePath   = "templates/index.html"
	commandTemplatePath = "templates/command.html"
)

var (
	port = flag.String("port", ":4901", "Server port")

	//go:embed templates/index.html
	indexTemplateFS embed.FS

	//go:embed templates/command.html
	commandTemplateFS embed.FS

	indexTemplate   *template.Template
	commandTemplate *template.Template
)

func init() {
	indexTemplate = template.Must(template.ParseFS(indexTemplateFS, indexTemplatePath))
	commandTemplate = template.Must(template.ParseFS(commandTemplateFS, commandTemplatePath))
}

type data struct {
	Title  string
	Output string
}

func main() {
	flag.Parse()

	log.Printf("CRUDD is starting up")

	ctx := context.Background()

	setupHandlers()

	httpServer := &http.Server{
		Addr:    *port,
		Handler: logging()(http.DefaultServeMux),
	}

	gracefulshutdown.AddShutdownHandler(func() error {
		log.Println("CRUDD is shutting down")
		return httpServer.Shutdown(ctx)
	})

	log.Println("CRUDD is ready to handle requests at", *port)
	if err := httpServer.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", *port, err)
		}
	}

	gracefulshutdown.WaitForShutdown()
	log.Println("CRUDD has been shut down")
}

func logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				ipAddress := r.RemoteAddr
				fwdAddress := r.Header.Get("X-Forwarded-For")
				if fwdAddress != "" {
					ipAddress = fwdAddress
				}
				log.Println(r.Method, r.URL.Path, ipAddress)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func setupHandlers() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", indexHandler)

	for _, command := range commandlib.Commands {
		http.HandleFunc(fmt.Sprintf("/%s", command.Name), createCommandHandler(command))
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := indexTemplate.Execute(w, map[string]interface{}{
		"commands": commandlib.Commands,
	}); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func createCommandHandler(command commandlib.Command) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		writeCommandOutput(w, runCommand(command.Path, command.Args))
	}
}

func writeCommandOutput(w http.ResponseWriter, d *data) {
	if err := commandTemplate.Execute(w, d); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func runCommand(bin, args string) *data {
	var cmd *exec.Cmd

	if args == "" {
		cmd = exec.Command(bin)
	} else {
		cmd = exec.Command(bin, strings.Split(args, " ")...)
	}

	log.Printf("Executing cmd: %s", cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return &data{
			Title:  fmt.Sprintf("%s %s", bin, args),
			Output: fmt.Sprintf("failed to run %s: %s: %s", bin, err, out),
		}
	}

	return &data{
		Title:  fmt.Sprintf("%s %s", bin, args),
		Output: string(out),
	}
}
