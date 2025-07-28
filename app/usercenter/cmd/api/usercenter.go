package main

import (
	"flag"
	"fmt"

	"document_agent/app/usercenter/cmd/api/internal/config"
	"document_agent/app/usercenter/cmd/api/internal/handler"
	"document_agent/app/usercenter/cmd/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/usercenter.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf,
		rest.WithCors("*"),
		rest.WithCorsHeaders("Content-Type", "Authorization"),
	)

	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
