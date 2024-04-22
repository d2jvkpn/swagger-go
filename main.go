package main

import (
	"context"
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

type Server struct {
	Release bool   `json:"release"`
	Address string `json:"address"`
	Path    string `json:"path"`
	TlsCert string `json:"tls_cert"`
	TlsKey  string `json:"tls_key"`

	listener net.Listener
	Engine   *gin.Engine `json:"-"`
	*http.Server
}

func main() {
	var (
		server Server
		err    error
		errch  chan error
		quit   chan os.Signal
		logger *slog.Logger
	)

	flag.BoolVar(&server.Release, "release", false, "run in release mode")
	flag.StringVar(&server.Address, "http.addr", ":3056", "http listening address")
	flag.StringVar(&server.Path, "http.path", "", "http base path")
	flag.StringVar(&server.TlsCert, "tls.cert", "", "http tls cert file")
	flag.StringVar(&server.TlsKey, "tls.key", "", "http tls key file")

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
			logger.Error("exit", "error", err)
			os.Exit(1)
		}
	}()

	if err = server.Setup(); err != nil {
		return
	}

	swagger_path := "/swagger"
	if server.Path != "" {
		swagger_path = "/" + server.Path + "/swagger"
	}
	server.Engine.NoRoute(func(ctx *gin.Context) {
		ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+swagger_path+"/index.html")
	})
	LoadSwagger(&server.Engine.RouterGroup)

	// engine.Run(http_addr)
	logger.Info("http server is up", "config", server)
	go func() {
		var e error

		e = server.Serve()
		// errors.Is(e, http.ErrServerClosed)
		// e != http.ErrServerClosed
		errch <- fmt.Errorf("server_closed")

		logger.Error("service has been shut down", "error", e)
	}()

	errch = make(chan error, 1) // the cap of the channel should be equal to number of services
	quit = make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // syscall.SIGUSR2

	//	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//	defer stop()
	//	go func() {
	//		e := srv.ListenAndServe()
	//		if e != nil && !errors.Is(e, http.ErrServerClosed) {
	//			logger.Error("listen and serve", "error", e)
	//		}
	//	}()
	//	<-ctx.Done()

	syncErrors := func(count int) {
		for i := 0; i < count; i++ {
			err = errors.Join(err, <-errch)
		}
	}

	count := cap(errch)
	select {
	case err = <-errch:
		logger.Error("... received from channel", "error", err)
		// shutdown other services

		count -= 1
	case sig := <-quit:
		// if sig == syscall.SIGUSR2 {...}
		// fmt.Fprintf(os.Stderr, "... received signal: %s\n", sig)

		logger.Warn("... quit", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		e := server.Shutdown(ctx)
		cancel()
		if e != nil {
			logger.Error("shutdown the server", "error", e)
		}
		// shutdown other services

		// errch <- fmt.Errorf("signal: %s", sig.String())
	}

	logger.Warn("sync errors", "count", count)
	syncErrors(count)
}

func (self *Server) Setup() (err error) {
	var (
		cert   tls.Certificate
		router *gin.RouterGroup
	)

	if self.listener, err = net.Listen("tcp", self.Address); err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	if self.Release {
		gin.SetMode(gin.ReleaseMode)
		self.Engine = gin.New()
	} else {
		self.Engine = gin.Default()
	}
	self.Engine.RedirectTrailingSlash = false
	router = &self.Engine.RouterGroup

	self.Path = strings.Trim(self.Path, "/")
	if self.Path != "" {
		*router = *(router.Group(self.Path))
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

	self.Server = new(http.Server)
	self.Server.Handler = self.Engine

	if self.TlsCert != "" && self.TlsKey != "" {
		if cert, err = tls.LoadX509KeyPair(self.TlsCert, self.TlsKey); err != nil {
			return
		}
		self.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	return nil
}

func (self *Server) Serve() (err error) {
	return self.Server.Serve(self.listener)
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

	for i := 0; i < len(updates); i++ {
		updates[i](docs.SwaggerInfo)
	}

	// router.GET("/", func(ctx *gin.Context) {
	// 	ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+"/swagger/index.html")
	// })

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

/*

//	example_01: Hello godoc
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
