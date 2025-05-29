package errors

import "google.golang.org/grpc/codes"

type AppError interface {
	RootError() error
	Message() string
	GrpcStatus() codes.Code
	Error() string
}
