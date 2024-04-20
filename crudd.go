package main

import (
	"bufio"
	"context"
	"crudd/commandlib"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"

	"github.com/MatthewLavine/gracefulshutdown"
)

const (
	indexTemplatePath         = "templates/index.html"
	commandHeaderTemplatePath = "templates/command_header.html"
	commandFooterTemplatePath = "templates/command_footer.html"
)

var (
	port = flag.String("port", ":4901", "Server port")

	//go:embed templates/index.html
	indexTemplateFS embed.FS

	//go:embed templates/command_header.html
	commandHeaderTemplateFS embed.FS

	//go:embed templates/command_footer.html
	commandFooterTemplateFS embed.FS

	//go:embed static
	staticFS embed.FS

	indexTemplate         *template.Template
	commandHeaderTemplate *template.Template
	commandFooterTemplate *template.Template
)

func init() {
	indexTemplate = template.Must(template.ParseFS(indexTemplateFS, indexTemplatePath))
	commandHeaderTemplate = template.Must(template.ParseFS(commandHeaderTemplateFS, commandHeaderTemplatePath))
	commandFooterTemplate = template.Must(template.ParseFS(commandFooterTemplateFS, commandFooterTemplatePath))
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
	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static FS root: %v", err)
	}
	fs := http.FileServer(http.FS(staticRoot))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", indexHandler)

	for _, command := range commandlib.Commands {
		http.HandleFunc(fmt.Sprintf("/%s", command.Name), createCommandHandler(command))
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := indexTemplate.Execute(w, map[string]interface{}{
		"existingCommands":         commandlib.ExistingCommands(),
		"nonExistingCommands":      commandlib.NonExistingCommands(),
		"countNonExistingCommands": len(commandlib.NonExistingCommands()),
	}); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func createCommandHandler(command commandlib.Command) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		writeCommandHeader(w, map[string]interface{}{
			"title": fmt.Sprintf("%s %s", command.Path, command.Args),
		})
		rc.Flush()
		writeOutputStreaming(w, rc, startCommandStreaming(command.Path, command.Args))
		rc.Flush()
		writeCommandFooter(w)
		rc.Flush()
	}
}

func writeCommandHeader(w http.ResponseWriter, data any) {
	if err := commandHeaderTemplate.Execute(w, data); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func writeCommandFooter(w http.ResponseWriter) {
	if err := commandFooterTemplate.Execute(w, nil); err != nil {
		fmt.Fprintf(w, "failed to execute template: %v", err)
		return
	}
}

func writeOutputStreaming(w http.ResponseWriter, rc *http.ResponseController, outputScanner *bufio.Scanner) {
	for outputScanner.Scan() {
		s := outputScanner.Text()
		fmt.Fprintln(w, s)
		rc.Flush()
		log.Printf("streamed %d bytes to client: %s", len(outputScanner.Bytes()), s)
	}
	if err := outputScanner.Err(); err != nil {
		log.Fatalf("failed to stream output: %v", err)
	}
}

func startCommandStreaming(bin, args string) *bufio.Scanner {
	var cmd *exec.Cmd

	if args == "" {
		cmd = exec.Command(bin)
	} else {
		cmd = exec.Command(bin, strings.Split(args, " ")...)
	}

	log.Printf("Executing cmd: %s", cmd)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		msg := fmt.Sprintf("failed to get stdout pipe for command %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg))
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		msg := fmt.Sprintf("failed to get stderr pipe for command %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg))
	}

	if err := cmd.Start(); err != nil {
		msg := fmt.Sprintf("failed to run %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg))
	}

	return bufio.NewScanner(io.MultiReader(stdOut, stdErr))
}
