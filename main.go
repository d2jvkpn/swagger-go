package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
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
	"github.com/spf13/viper"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

var (
	build_time      string
	build_host      string
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

	listener     net.Listener
	Engine       *gin.Engine `json:"-"`
	*http.Server `json:"-"`
}

func main() {
	var (
		config string
		err    error
		errch  chan error
		quit   chan os.Signal
		logger *slog.Logger
		server Server

		data SwaggerConfig
	)

	flag.BoolVar(&server.Release, "release", false, "run in release mode")
	flag.StringVar(&server.Address, "http.addr", ":3066", "http listening address")
	flag.StringVar(&server.Path, "http.path", "", "http base path")
	flag.StringVar(&server.TlsCert, "tls.cert", "", "http tls cert file")
	flag.StringVar(&server.TlsKey, "tls.key", "", "http tls key file")
	flag.StringVar(&config, "config", "", "yaml configuration file path")

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
		fmt.Fprintf(output, "build_host: %s\n", build_host)
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
		} else {
			logger.Info("exit")
		}
	}()

	// 1. config
	data = NewSwaggerConfig()
	if err = SetAccounts(config, &data); err != nil {
		return
	}

	// 2. server
	if err = server.Setup(data.Accounts); err != nil {
		return
	}

	// 3
	swagger_path := "/swagger"
	if server.Path != "" {
		swagger_path = "/" + server.Path + "/swagger"
	}
	server.Engine.NoRoute(func(ctx *gin.Context) {
		ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+swagger_path+"/index.html")
	})

	// 2.2
	go_version := runtime.Version()
	meta := map[string]*string{
		"build_time":      &build_time,
		"build_host":      &build_host,
		"go_version":      &go_version,
		"git_repository":  &git_repository,
		"git_branch":      &git_branch,
		"git_commit_id":   &git_commit_id,
		"git_commit_time": &git_commit_time,
		"git_tree_state":  &git_tree_state,
	}

	meta_bts, _ := json.Marshal(meta)
	server.Engine.RouterGroup.GET("/meta", func(ctx *gin.Context) {
		// ctx.JSON(http.StatusOK, meta)

		ctx.Header("Content-Type", "application/json")
		ctx.Writer.Write(meta_bts)
	})

	// 4
	LoadSwagger(&server.Engine.RouterGroup)

	errch = make(chan error, 1) // the cap of the channel should be equal to number of services
	quit = make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // syscall.SIGUSR2

	//	link: https://dev.to/antonkuklin/golang-graceful-shutdown-3n6d
	//
	//	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//	defer stop()
	//	go func() {
	//		e := srv.ListenAndServe()
	//		if e != nil && !errors.Is(e, http.ErrServerClosed) {
	//			logger.Error("listen and serve", "error", e)
	//		}
	//	}()
	//	<-ctx.Done()

	// engine.Run(http_addr)
	logger.Info("http server is up", "config", server)
	go func() {
		var e error

		e = server.Serve()
		// e != http.ErrServerClosed
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			errch <- e
		} else {
			errch <- nil
		}

		logger.Error("service has been shut down", "error", e)
	}()

	syncErrors := func(count int) {
		logger.Warn("sync errors", "count", count)
		for i := 0; i < count; i++ {
			err = errors.Join(err, <-errch)
		}
	}

	select {
	case err = <-errch:
		logger.Error("... received from channel errch")
		// shutdown other services

		syncErrors(cap(errch) - 1)
	case sig := <-quit:
		logger.Warn("... received from channel quit", "signal", sig.String())
		// if sig == syscall.SIGUSR2 {...} // works on linux only

		// 1. shutdow http server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e := server.Shutdown(ctx)
		cancel()
		if e != nil {
			logger.Error("shutdown the server", "error", e)
		}

		// 2. shutdown other services

		// errch <- fmt.Errorf("signal: %s", sig.String())
		syncErrors(cap(errch))
	}
}

func SetAccounts(config string, data *SwaggerConfig) (err error) {
	if config == "" {
		return nil
	}

	field := "swagger" // subfield of config
	if p1, p2, ok := strings.Cut(config, "::"); ok {
		config, field = p1, p2
	}

	vp := viper.New()
	vp.SetConfigType("yaml")

	vp.SetConfigName("")
	vp.SetConfigFile(config)

	if err = vp.ReadInConfig(); err != nil {
		return fmt.Errorf("ReadInConfig: %w", err)
	}

	if vp.Sub(field) == nil {
		return fmt.Errorf("no subfield %q in config", field)
	}

	// fmt.Printf("~~~ %s, %s, %+v\n", config, field, vp)
	if err = vp.Sub(field).Unmarshal(data); err != nil {
		return fmt.Errorf("Viper.Unmarshal: %w", err)
	}
	if len(data.Accounts) == 0 {
		return fmt.Errorf("no accounts in config")
	}

	return nil
}

func (self *Server) Setup(accounts []Account) (err error) {
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

	if len(accounts) > 0 {
		self.Engine.Use(BasicAuth(accounts))
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

type Account struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type SwaggerConfig struct {
	Accounts []Account `mapstructure:"accounts"`
}

func NewSwaggerConfig() SwaggerConfig {
	return SwaggerConfig{Accounts: make([]Account, 0)}
}

// handle key: no_token, invalid_token, incorrect_token, User:XXXX
func BasicAuth(accounts []Account) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var (
			username string
			password string
			found    bool
			bts      []byte
			err      error
		)

		key := ctx.GetHeader("Authorization")
		unauth := func(kind string) {
			ctx.Header("Www-Authenticate", `Basic realm="login required"`)
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": "Unauthorized", "kind": kind})
			ctx.Abort()
		}

		if !strings.HasPrefix(key, "Basic ") {
			unauth("no_token")
			return
		}

		if bts, err = base64.StdEncoding.DecodeString(key[6:]); err != nil {
			unauth("invalid_token")
			return
		}

		if username, password, found = strings.Cut(string(bts), ":"); !found {
			unauth("invalid_token")
			return
		}

		for i := range accounts {
			if accounts[i].Username == username {
				if accounts[i].Password == password {
					// handle(ctx, fmt.Sprintf("User:%s", username))
					ctx.Next()
					return
				} else {
					unauth("incorrect_account")
					return
				}
			}
		}

		unauth("incorrect_account")
	}
}

/*

// e01: Hello godoc
//
// @Summary		Show an account
// @Description	get string by ID
// @Tags			accounts
// @Accept			json
// @Produce		json
// @Param	id		path	int		true	"Account ID"
// @Param	name	query	string	flase	"Account Name"
// @Param	request	body	Login	true	"user password"
// @Success		200	{object}	LoginResponse
// @Failure		400	{object}	error
// @Failure		404	{object}	error
// @Failure		500	{object}	error
// @Router			/accounts/{id}	[get]
func Hello(ctx *gin.Context) {
	// TODO: ...
}

// User Login Body
//
// @description	phone, email, name
// @description	xxxx
type Login struct {
	// User Email address
	Email    string `json:"email,omitempty" example:"john@noreply.local"`
	Name     string `json:"name,omitempty" example:"John Doe" minLength:"2" maxLength:"24"`
	Age      int    `json:"age" example:"2" minimum:"1" maximum:"20"`
	Role     string `json:"role,omitempty" enums:"admin, maintainer, owner" binding:"required,oneof=admin maintainer owner"`
	Password string `json:"password" example:"acbABC123" minLength:"8" maxLength:"24"`
	Date string `json:"date,omitempty" binding:"required" time_format:"2006-01-02"`

	// Option<map[string]any>: response data
	Data map[string]any `json:"data,omitempty" swaggertype:"object,string" example:"ans:hello,value:42"`
	XX string `swaggerignore:"true"`
}

*/
