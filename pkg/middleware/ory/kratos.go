package ory

import (
	"context"
	"fmt"
	"step/internal/conf"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	client "github.com/ory/kratos-client-go"
)

// 解析token
func Token(c *conf.Data) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var new_ctx context.Context
			if tr, ok := transport.FromServerContext(ctx); ok {
				// 断言成HTTP的Transport可以拿到特殊信息
				if ht, ok := tr.(*http.Transport); ok {
					a := ht.RequestHeader().Get("Authorization")
					if a == "" {
						return "", fmt.Errorf("token is missing")
					}

					parts := strings.Split(a, " ")
					if len(parts) != 2 || parts[0] != "Bearer" {
						return "", fmt.Errorf("authorization header is invalid")
					}

					token := parts[1]
					configuration := client.NewConfiguration()
					configuration.Servers = []client.ServerConfiguration{
						{
							URL: "c.Ory.KratosPublicUrl", // Kratos Public API
						},
					}
					apiClient := client.NewAPIClient(configuration)
					cookie := "ory_kratos_session=" + token
					session, _, err := apiClient.FrontendAPI.ToSession(ctx).Cookie(cookie).Execute()
					if err != nil {
						return "", fmt.Errorf("token is invalid: %v", err)
					}
					uid := session.Identity.Id
					fmt.Println("got uid: ", uid)
					new_ctx = context.WithValue(ctx, "uid", uid)

					ht.ReplyHeader().Set("Access-Control-Allow-Origin", "*")
					ht.ReplyHeader().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,PUT,PATCH,DELETE")
					ht.ReplyHeader().Set("Access-Control-Allow-Credentials", "true")
					ht.ReplyHeader().Set("Access-Control-Allow-Headers", "Content-Type,"+
						"X-Requested-With,Access-Control-Allow-Credentials,User-Agent,Content-Length,Authorization")
				}
			} else {
				return "", fmt.Errorf("token is missing")
			}
			return handler(new_ctx, req)
		}
	}
}
