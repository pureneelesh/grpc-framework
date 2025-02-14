/*
Package server is the gRPC implementation of the SDK gRPC server
Copyright 2018 Portworx

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package server

import (
	"context"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/libopenstorage/grpc-framework/pkg/auth"
	"github.com/libopenstorage/grpc-framework/pkg/auth/role"
	"github.com/rs/cors"
	"google.golang.org/grpc"

	"golang.org/x/time/rate"
)

// TLSConfig points to the cert files needed for HTTPS
type TLSConfig struct {
	// CertFile is the path to the cert file
	CertFile string
	// KeyFile is the path to the key file
	KeyFile string
}

// SecurityConfig provides configuration for SDK auth
type SecurityConfig struct {
	// Role implementation
	Role role.RoleManager
	// Tls configuration
	Tls *TLSConfig
	// Authenticators per issuer. You can register multple authenticators
	// based on the "iss" string in the string. For example:
	// map[string]auth.Authenticator {
	//     "https://accounts.google.com": googleOidc,
	//     "openstorage-sdk-auth: selfSigned,
	// }
	Authenticators map[string]auth.Authenticator
}

type RestServerPrometheusConfig struct {
	Enabled bool

	// Defaults to `/metrics` if not provided
	Path string
}

type RestServerCorsConfig struct {
	Enabled bool

	// If not set, the framework will set up the cors
	CustomOptions *cors.Options
}

type RestServerConfig struct {
	Enabled          bool
	Port             string
	CorsOptions      RestServerCorsConfig
	PrometheusConfig RestServerPrometheusConfig
}

type RateLimiterConfig struct {
	RateLimiter        RateLimiter
	RateLimiterPerUser RateLimiter
}

// ServerConfig provides the configuration to the SDK server
type ServerConfig struct {
	// Name of the server
	Name string
	// Net is the transport for gRPC: unix, tcp, etc.
	// Defaults to `tcp` if the value is not provided.
	Net string
	// Address is the port number or the unix domain socket path.
	// For the gRPC Server. This value goes together with `Net`.
	Address string
	// REST server configuration
	RestConfig RestServerConfig
	// Unix domain socket for local communication. This socket
	// will be used by the REST Gateway to communicate with the gRPC server.
	// Only set for testing. Having a '%s' can be supported to use the
	// name of the driver as the driver name.
	Socket string
	// (optional) Location for audit log.
	// If not provided, it will go to /var/log/openstorage-audit.log
	AuditOutput io.Writer
	// (optional) Location of access log.
	// This is useful when authorization is not running.
	// If not provided, it will go to /var/log/grpc-framework-access.log
	AccessOutput io.Writer
	// Security configuration
	Security *SecurityConfig
	// RateLimiters provide caller with the ability to setup rate limits for
	// the gRPC server
	RateLimiters RateLimiterConfig
	// ServerExtensions allows you to extend the SDK gRPC server
	// with callback functions that are sequentially executed
	// at the end of Server.Start()
	//
	// To add your own service to the SDK gRPC server,
	// just append a function callback that registers it:
	//
	// s.config.ServerExtensions = append(s.config.ServerExtensions,
	// 		func(gs *grpc.Server) {
	//			api.RegisterCustomService(gs, customHandler)
	//		})
	GrpcServerExtensions []func(grpcServer *grpc.Server)

	// RestServerExtensions allows for extensions to be added
	// to the SDK Rest Gateway server.
	//
	// To add your own service to the SDK REST Server, simply add your handlers
	// to the RestSererExtensions slice. These handlers will be registered on the
	// REST Gateway http server.
	RestServerExtensions []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error

	// UnaryServerInterceptors will be interceptors added to the end of the default chain
	UnaryServerInterceptors []grpc.UnaryServerInterceptor

	// StreamServerInterceptors will be interceptors added to the end of the default chain
	StreamServerInterceptors []grpc.StreamServerInterceptor

	// ServerOptions hold any special gRPC server options
	ServerOptions []grpc.ServerOption
}

var (
	DefaultRestServerCors = cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "HEAD", "PUT", "OPTIONS"},
		AllowCredentials: true,
	}
	DefaultRateLimiter        = rate.NewLimiter(100, 50)
	DefaultRateLimiterPerUser = rate.NewLimiter(10, 25)
)

func (c *ServerConfig) RegisterGrpcServers(handlers func(grpcServer *grpc.Server)) *ServerConfig {
	if c == nil {
		return c
	}
	c.GrpcServerExtensions = append(c.GrpcServerExtensions, handlers)
	return c
}

func (c *ServerConfig) RegisterRestHandlers(
	handlers ...func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error,
) *ServerConfig {
	if c == nil {
		return c
	}
	c.RestServerExtensions = append(c.RestServerExtensions, handlers...)
	return c
}

func (c *ServerConfig) WithRestCors(co cors.Options) *ServerConfig {
	if c == nil {
		return c
	}
	c.RestConfig.CorsOptions.Enabled = true
	c.RestConfig.CorsOptions.CustomOptions = &co
	return c
}

func (c *ServerConfig) WithRestPrometheus(path string) *ServerConfig {
	if c == nil {
		return c
	}
	c.RestConfig.PrometheusConfig.Enabled = true
	c.RestConfig.PrometheusConfig.Path = path
	return c
}

func (c *ServerConfig) WithDefaultRestServer(port string) *ServerConfig {
	if c == nil {
		return c
	}

	c.RestConfig.Port = port
	c.RestConfig.Enabled = true
	return c.WithRestCors(DefaultRestServerCors).WithRestPrometheus("/metrics")
}

func (c *ServerConfig) WithServerUnaryInterceptors(i ...grpc.UnaryServerInterceptor) *ServerConfig {
	if c == nil {
		return c
	}

	c.UnaryServerInterceptors = append(c.UnaryServerInterceptors, i...)
	return c
}

func (c *ServerConfig) WithServerStreamInterceptors(i ...grpc.StreamServerInterceptor) *ServerConfig {
	if c == nil {
		return c
	}

	c.StreamServerInterceptors = append(c.StreamServerInterceptors, i...)
	return c
}

func (c *ServerConfig) WithServerOptions(opt ...grpc.ServerOption) *ServerConfig {
	if c == nil {
		return c
	}

	c.ServerOptions = append(c.ServerOptions, opt...)
	return c
}

func (c *ServerConfig) WithRateLimiter(r RateLimiter) *ServerConfig {
	if c == nil {
		return c
	}

	c.RateLimiters.RateLimiter = r
	return c
}

func (c *ServerConfig) WithRateLimiterPerUser(r RateLimiter) *ServerConfig {
	if c == nil {
		return c
	}

	c.RateLimiters.RateLimiterPerUser = r
	return c
}

func (c *ServerConfig) WithDefaultRateLimiters() *ServerConfig {
	return c.
		WithRateLimiter(DefaultRateLimiter).
		WithRateLimiterPerUser(DefaultRateLimiterPerUser)
}
