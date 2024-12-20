package main

import (
	"bufio"
	"context"
	"crudd/commandlib"
	"embed"
	"flag"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/MatthewLavine/gracefulshutdown"
)

const (
	indexTemplatePath         = "templates/index.html"
	commandHeaderTemplatePath = "templates/command_header.html"
	commandFooterTemplatePath = "templates/command_footer.html"
)

var (
	port           = flag.String("port", ":4901", "Server port")
	testFSRoot     = flag.String("test_fs_root", "", "Fake FS root for testing")
	verboseLogging = flag.Bool("verbose", false, "Verbose output")

	//go:embed templates
	templateFS embed.FS

	//go:embed static
	staticFS embed.FS

	indexTemplate         *template.Template
	commandHeaderTemplate *template.Template
	commandFooterTemplate *template.Template
)

func init() {
	indexTemplate = template.Must(template.ParseFS(templateFS, indexTemplatePath))
	commandHeaderTemplate = template.Must(template.ParseFS(templateFS, commandHeaderTemplatePath))
	commandFooterTemplate = template.Must(template.ParseFS(templateFS, commandFooterTemplatePath))
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
		startTime := time.Now()
		scanner, readerDoneChan, exitCodeChan := startCommandStreaming(r.Context(), command.Path, command.Args)
		writeOutputStreaming(w, rc, scanner, readerDoneChan)
		rc.Flush()
		exitCode := <-exitCodeChan
		latency := time.Since(startTime)
		log.Printf("Command took %s to run and exited with code %d", latency.Round(time.Microsecond), exitCode)
		fmt.Fprintf(w, "\nCommand exited with code: %d", exitCode)
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

func writeOutputStreaming(w http.ResponseWriter, rc *http.ResponseController, outputScanner *bufio.Scanner, readerDoneChan chan struct{}) {
	defer func() {
		readerDoneChan <- struct{}{}
	}()
	for outputScanner.Scan() {
		s := outputScanner.Text()
		fmt.Fprintln(w, html.EscapeString(s))
		rc.Flush()
		if *verboseLogging {
			log.Printf("Streamed %d bytes to client: %s", len(outputScanner.Bytes()), s)
		}
	}
	if err := outputScanner.Err(); err != nil {
		log.Printf("failed to stream output: %v", err)
	}
}

func startCommandStreaming(ctx context.Context, bin, args string) (*bufio.Scanner, chan struct{}, chan int) {
	var cmd *exec.Cmd
	readerDoneChan := make(chan struct{}, 1)
	exitCodeChan := make(chan int, 1)

	if *testFSRoot != "" {
		log.Printf("testFSRoot was set to: %v", *testFSRoot)
		bin = *testFSRoot + bin
	}

	if windowsTestFS := strings.Contains(*testFSRoot, "\\"); windowsTestFS {
		bin = strings.ReplaceAll(bin, "/", "\\")
	}

	if args == "" {
		cmd = exec.CommandContext(ctx, bin)
	} else {
		cmd = exec.CommandContext(ctx, bin, strings.Split(args, " ")...)
	}

	cmd.Cancel = func() error {
		_ = cmd.Process.Kill() // intentionally ignore error because process may already be dead
		return nil
	}

	cmd.WaitDelay = time.Duration(5) * time.Second

	log.Printf("Executing cmd: %s", cmd)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		msg := fmt.Sprintf("failed to get stdout pipe for command %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg)), readerDoneChan, exitCodeChan
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		msg := fmt.Sprintf("failed to get stderr pipe for command %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg)), readerDoneChan, exitCodeChan
	}

	if err := cmd.Start(); err != nil {
		msg := fmt.Sprintf("failed to run %s: %s", bin, err)
		log.Print(msg)
		return bufio.NewScanner(strings.NewReader(msg)), readerDoneChan, exitCodeChan
	}

	go func(cmd *exec.Cmd) {
		select {
		case <-readerDoneChan:
		case <-ctx.Done():
			log.Println("Request cancelled early")
		}
		// Call cmd.Wait to ensure we release file descriptors but ignore errors because Wait
		// races with the request context and can cause spurious errors here.
		_ = cmd.Wait()
		exitCodeChan <- cmd.ProcessState.ExitCode()
	}(cmd)

	return bufio.NewScanner(io.MultiReader(stdOut, stdErr)), readerDoneChan, exitCodeChan
}
