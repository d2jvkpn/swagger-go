package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"swagger-go/docs"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

var (
	BUILD_Time     string
	GIT_Repository string
	GIT_Branch     string
	GIT_CommitId   string
	GIT_CommitTime string
)

func main() {
	var (
		release           bool
		http_addr         string
		http_path         string
		tls_cert          string
		tls_key           string
		swagger_title     string
		swagger_host      string
		swagger_base_path string

		err    error
		errch  chan error
		quit   chan os.Signal
		logger *slog.Logger
		cert   tls.Certificate

		httpListener net.Listener
		router       *gin.RouterGroup
		engine       *gin.Engine
		server       *http.Server
	)

	flag.BoolVar(&release, "release", false, "run in release mode")

	flag.StringVar(&http_addr, "http.addr", ":3056", "http listening address")
	flag.StringVar(&http_path, "http.path", "", "http base path")
	flag.StringVar(&tls_cert, "tls.cert", "", "http tls key cert")
	flag.StringVar(&tls_key, "tls.key", "", "http tls key file")

	flag.StringVar(&swagger_title, "swagger.title", "Swagger Example API", "swagger title")
	flag.StringVar(&swagger_host, "swagger.host", "petstore.swagger.io", "swagger host")
	flag.StringVar(&swagger_base_path, "swagger.base-path", "/app/v1", "swagger base path")

	flag.Usage = func() {
		output := flag.CommandLine.Output()

		fmt.Fprintf(output, "# swagger-go (https://github.com/d2jvkpn/swagger-go)\n")
		fmt.Fprintf(output, "\n#### Usage\n```text\n")
		flag.PrintDefaults()
		fmt.Fprintf(output, "```\n")

		fmt.Fprintf(output, "\n#### Build\n```yaml\n")
		fmt.Fprintf(output, "build_time: %s\n", BUILD_Time)
		fmt.Fprintf(output, "git_repository: %s\n", GIT_Repository)
		fmt.Fprintf(output, "git_branch: %s\n", GIT_Branch)
		fmt.Fprintf(output, "git_commit_id: %s\n", GIT_CommitId)
		fmt.Fprintf(output, "git_commit_time: %s\n", GIT_CommitTime)
		fmt.Fprintf(output, "```\n")
	}

	flag.Parse()

	// logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))

	defer func() {
		if err != nil {
			os.Exit(1)
		}
	}()

	if httpListener, err = net.Listen("tcp", http_addr); err != nil {
		err = fmt.Errorf("net.Listen: %w", err)
		return
	}

	if release {
		gin.SetMode(gin.ReleaseMode)
		engine = gin.New()
	} else {
		engine = gin.Default()
	}
	engine.RedirectTrailingSlash = false
	router = &engine.RouterGroup

	if http_path != "" {
		*router = *(router.Group(http_path))
	}

	LoadSwagger(router, func(spec *swag.Spec) {
		spec.Title = swagger_title
		spec.Host = swagger_host
		spec.BasePath = swagger_base_path
	})

	// engine.Run(addr)

	server = new(http.Server)

	if tls_cert != "" && tls_key != "" {
		if cert, err = tls.LoadX509KeyPair(tls_cert, tls_key); err != nil {
			return
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	go func() {
		var err error

		if err = server.Serve(httpListener); err != http.ErrServerClosed {
			errch <- fmt.Errorf("http_server_down")
		}
	}()

	errch = make(chan error, 1)
	quit = make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // syscall.SIGUSR2

	syncErrors := func(num int) {
		for i := 0; i < num; i++ {
			err = errors.Join(err, <-errch)
		}
	}

	select {
	case err = <-errch:
		syncErrors(cap(errch) - 1)

		logger.Error("... received from error channel", "error", err)
	case sig := <-quit:
		// if sig == syscall.SIGUSR2 {...}
		// fmt.Fprintf(os.Stderr, "... received signal: %s\n", sig)
		errch <- fmt.Errorf("shutdown")
		syncErrors(cap(errch))

		logger.Info("... quit", "signal", sig.String(), "error", err)
	}
}

func LoadSwagger(router *gin.RouterGroup, alert ...func(*swag.Spec)) {
	// programmatically set swagger info
	docs.SwaggerInfo.Title = "Swagger Example API"
	docs.SwaggerInfo.Description = "This is a sample server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "petstore.swagger.io"
	docs.SwaggerInfo.BasePath = "/v2"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	if len(alert) > 0 {
		alert[0](docs.SwaggerInfo)
	}

	/*
		// "/swagger"
		router.GET("/", func(ctx *gin.Context) {
			ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+"/index.html")
		})
	*/

	// "/swagger/*any"
	router.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
