package main

import (
	"flag"
	"fmt"
	"net/http"

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
		addr              string
		http_path         string
		swagger_title     string
		swagger_host      string
		swagger_base_path string

		router *gin.RouterGroup
		engine *gin.Engine
	)

	flag.BoolVar(&release, "release", false, "run in release mode")
	flag.StringVar(&addr, "addr", ":3056", "http listening address")
	flag.StringVar(&http_path, "http.path", "", "http base path")

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

	engine.Run(addr)
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

	// "/swagger"
	router.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+"/index.html")
	})

	// "/swagger/*any"
	router.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
