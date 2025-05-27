package server

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"net"
	"weaveTest/internal/proto/generated"
	"weaveTest/internal/server/github"
)

const (
	DefaultPort = ":50051"
)

type Server struct {
	generated.GithubSearchServiceServer
	port   string
	logger zap.Logger
	sock   net.Listener
	server *grpc.Server
}

// NewServer creates and returns a new Server instance configured with the provided context, port, and logger.
// If the port is empty it defaults to the predefined DefaultPort.
func NewServer(port string, logger zap.Logger) *Server {
	if port == "" {
		port = DefaultPort
	}
	logger.Debug("initializing tcp listener")
	sock, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("failed to listen on port", zap.String("port", port), zap.Error(err))
		panic(err)
	}
	logger.Debug("initializing grpcServer")
	grpcServer := grpc.NewServer()

	ref := &Server{
		sock:   sock,
		server: grpcServer,
		port:   port,
		logger: logger,
	}
	generated.RegisterGithubSearchServiceServer(grpcServer, ref)
	return ref
}

// NewServerWithDefaultPort creates a new Server instance with the default port
func NewServerWithDefaultPort(logger *zap.Logger) *Server {
	return NewServer("", *logger)
}

func (s *Server) Search(ctx context.Context, in *generated.SearchRequest) (*generated.SearchResponse, error) {
	s.logger.Debug("beginning search, args: ", zap.Dict("request", zap.String("name", in.SearchTerm), zap.String("user", in.User)))
	data, err := github.NewDataFetcher(s.logger).Fetch(ctx, in)
	if err != nil {
		grpcError := status.Error(err.GrpcStatus(), err.Message())
		// TODO: check if we can change the proto for descriptive errors
		return nil, grpcError
	}
	s.logger.Debug("successfully fetched data from remote")
	return data, nil
}

func (s *Server) Start() {
	err := s.server.Serve(s.sock)
	if err != nil {
		s.logger.Error("failed to start server", zap.Error(err))
		panic(err)
	}
	s.logger.Debug("server started", zap.String("port", s.port))
}

func (s *Server) Close() {
	s.logger.Debug("closing server")
	s.server.GracefulStop()
	err := s.sock.Close()
	if err != nil {
		return
	}
}
