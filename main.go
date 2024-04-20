package main

import (
	// "fmt"
	"flag"

	"swagger-go/docs"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

func main() {
	var (
		release           bool
		addr              string
		base_path         string
		swagger_title     string
		swagger_host      string
		swagger_base_path string

		router *gin.RouterGroup
		engine *gin.Engine
	)

	flag.BoolVar(&release, "release", false, "run in release mode")
	flag.StringVar(&addr, "addr", ":3056", "http listening address")
	flag.StringVar(&base_path, "base_path", "", "http base path")

	flag.StringVar(&swagger_title, "swagger.title", "Swagger Example API", "swagger title")
	flag.StringVar(&swagger_host, "swagger.host", "petstore.swagger.io", "swagger host")
	flag.StringVar(&swagger_base_path, "swagger.base-path", "/app/v1", "swagger base path")

	flag.Parse()

	if release {
		gin.SetMode(gin.ReleaseMode)
		engine = gin.New()
	} else {
		engine = gin.Default()
	}
	engine.RedirectTrailingSlash = false
	router = &engine.RouterGroup

	if base_path != "" {
		*router = *(router.Group(base_path))
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

	/*
		router.GET("/swagger", func(ctx *gin.Context) {
			ctx.Redirect(http.StatusTemporaryRedirect, ctx.FullPath()+"/index.html")
		})

		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	*/

	router.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
