package main

import (
	"crypto/tls"
	"encoding/json"
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
	"time"

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

type HttpConfig struct {
	Release bool   `json:"release"`
	Address string `json:"address"`
	Path    string `json:"path"`
	TlsCert string `json:"tls_cert"`
	TlsKey  string `json:"tls_key"`

	Listener net.Listener `json:"-"`
}

func main() {
	var (
		config HttpConfig
		err    error
		errch  chan error
		quit   chan os.Signal
		logger *slog.Logger
		server *http.Server
	)

	flag.BoolVar(&config.Release, "release", false, "run in release mode")
	flag.StringVar(&config.Address, "http.addr", ":3056", "http listening address")
	flag.StringVar(&config.Path, "http.path", "", "http base path")
	flag.StringVar(&config.TlsCert, "tls.cert", "", "http tls cert file")
	flag.StringVar(&config.TlsKey, "tls.key", "", "http tls key file")

	flag.StringVar(
		&docs.SwaggerInfo.Title, "swagger.title",
		docs.SwaggerInfo.Title, "swagger title",
	)

	flag.StringVar(&docs.SwaggerInfo.Host, "swagger.host", docs.SwaggerInfo.Host, "swagger host")
	flag.StringVar(&docs.SwaggerInfo.BasePath, "swagger.base-path", "/app/v1", "swagger base path")

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

	if server, err = NewServer(&config); err != nil {
		return
	}

	logger.Info("http server is up", "config", config)
	go func() {
		var err error

		if err = server.Serve(config.Listener); err != http.ErrServerClosed {
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

func NewServer(config *HttpConfig) (server *http.Server, err error) {
	var (
		cert   tls.Certificate
		router *gin.RouterGroup
		engine *gin.Engine
	)

	if config.Listener, err = net.Listen("tcp", config.Address); err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	if config.Release {
		gin.SetMode(gin.ReleaseMode)
		engine = gin.New()
	} else {
		engine = gin.Default()
	}
	engine.RedirectTrailingSlash = false
	router = &engine.RouterGroup

	config.Path = strings.Trim(config.Path, "/")
	if config.Path != "" {
		*router = *(router.Group(config.Path))
	}

	startup_time := time.Now().Format(time.RFC3339)
	go_version := runtime.Version()
	meta := map[string]*string{
		"build_time":      &build_time,
		"go_version":      &go_version,
		"git_repository":  &git_repository,
		"git_branch":      &git_branch,
		"git_commit_id":   &git_commit_id,
		"git_commit_time": &git_commit_time,
		"git_tree_state":  &git_tree_state,

		"startup_time": &startup_time,
	}

	meta_bts, _ := json.Marshal(meta)
	router.GET("/meta", func(ctx *gin.Context) {
		// ctx.JSON(http.StatusOK, meta)

		ctx.Header("Content-Type", "application/json")
		ctx.Writer.Write(meta_bts)
	})

	LoadSwagger(router)
	// engine.Run(http_addr)

	swagger_path := "/swagger"
	if config.Path != "" {
		swagger_path = "/" + config.Path + "/swagger"
	}
	engine.NoRoute(func(ctx *gin.Context) {
		ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+swagger_path+"/index.html")
	})

	server = new(http.Server)
	server.Handler = engine

	if config.TlsCert != "" && config.TlsKey != "" {
		if cert, err = tls.LoadX509KeyPair(config.TlsCert, config.TlsKey); err != nil {
			return
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	return server, nil
}

func LoadSwagger(router *gin.RouterGroup, updates ...func(*swag.Spec)) {
	// programmatically set swagger info
	if docs.SwaggerInfo.Title == "" {
		docs.SwaggerInfo.Title = "Swagger Example API"
	}

	docs.SwaggerInfo.Description = "This is a sample server."
	docs.SwaggerInfo.Version = "1.0"

	if docs.SwaggerInfo.Host == "" {
		docs.SwaggerInfo.Host = "petstore.swagger.io"
	}

	if docs.SwaggerInfo.BasePath == "" {
		docs.SwaggerInfo.BasePath = "/v2"
	}

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
