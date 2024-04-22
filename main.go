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
	"runtime"
	"strings"
	"syscall"

	"swagger-go/docs"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

var (
	build_time      string
	git_repository  string
	git_branch      string
	git_commit_id   string
	git_commit_time string
	git_tree_state  string
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
	flag.StringVar(&tls_cert, "tls.cert", "", "http tls cert file")
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
		fmt.Fprintf(output, "build_time: %s\n", build_time)
		fmt.Fprintf(output, "go_version: %s\n", runtime.Version())
		fmt.Fprintf(output, "git_repository: %s\n", git_repository)
		fmt.Fprintf(output, "git_branch: %s\n", git_branch)
		fmt.Fprintf(output, "git_commit_id: %s\n", git_commit_id)
		fmt.Fprintf(output, "git_commit_time: %s\n", git_commit_time)
		fmt.Fprintf(output, "git_tree_state: %s\n", git_tree_state)
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

	http_path = strings.Trim(http_path, "/")
	if http_path != "" {
		*router = *(router.Group(http_path))
	}

	meta := map[string]string{
		"build_time":      build_time,
		"go_version":      runtime.Version(),
		"git_repository":  git_repository,
		"git_branch":      git_branch,
		"git_commit_id":   git_commit_id,
		"git_commit_time": git_commit_time,
		"git_tree_state":  git_tree_state,
	}

	router.GET("/meta", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, meta)
	})

	LoadSwagger(router, func(spec *swag.Spec) {
		if swagger_title != "" {
			spec.Title = swagger_title
		}

		if swagger_host != "" {
			spec.Host = swagger_host
		}

		if swagger_base_path != "" {
			spec.BasePath = swagger_base_path
		}
	})

	// engine.Run(http_addr)

	swagger_path := "/swagger"
	if http_path != "" {
		swagger_path = "/" + http_path + "/swagger"
	}
	engine.NoRoute(func(ctx *gin.Context) {
		ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+swagger_path+"/index.html")
	})

	server = new(http.Server)
	server.Handler = engine

	if tls_cert != "" && tls_key != "" {
		if cert, err = tls.LoadX509KeyPair(tls_cert, tls_key); err != nil {
			return
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	logger.Info("http server is up", "http_addr", http_addr, "release", release)
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

func LoadSwagger(router *gin.RouterGroup, updates ...func(*swag.Spec)) {
	// programmatically set swagger info
	docs.SwaggerInfo.Title = "Swagger Example API"
	docs.SwaggerInfo.Description = "This is a sample server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "petstore.swagger.io"
	docs.SwaggerInfo.BasePath = "/v2"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	if len(updates) > 0 {
		updates[0](docs.SwaggerInfo)
	}

	// router.GET("/", func(ctx *gin.Context) {
	// 	ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+"/swagger/index.html")
	// })

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

/*

// example_01: Hello godoc
//
//	@Summary		Show an account
//	@Description	get string by ID
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param	id		path	int			true	"Account ID"
//	@Param	name	query	string		flase	"Account Name"
//	@Param	login	body	LoginUser	true	"user password"
//	@Success		200	{object}	Response
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Router			/accounts/{id}	[get]
func Hello(ctx *gin.Context) {
	// TODO: ...
}

*/
