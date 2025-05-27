package errors

import "google.golang.org/grpc/codes"

type ErrorKind int

func (e ErrorKind) ToGrpcStatus() codes.Code {
	return errorKindToGrpcStatusCode[e]
}

const (
	InvalidQuery ErrorKind = iota
	InvalidUser
	InvalidToken
	RateLimited
	Unknown
	InvalidResponse
	InvalidRequest
	InvalidResponseFormat
	InvalidRequestFormat
	Internal
	InvalidJSONBody
	InvalidHttpClient
	PaginationFailed
)

var errorKindToGrpcStatusCode = make(map[ErrorKind]codes.Code)

func init() {
	initErrorKindMap()
}

func initErrorKindMap() {
	errorKindToGrpcStatusCode[InvalidQuery] = codes.InvalidArgument
	errorKindToGrpcStatusCode[InvalidUser] = codes.InvalidArgument
	errorKindToGrpcStatusCode[InvalidToken] = codes.InvalidArgument
	errorKindToGrpcStatusCode[RateLimited] = codes.ResourceExhausted
	errorKindToGrpcStatusCode[Unknown] = codes.Unknown
	errorKindToGrpcStatusCode[InvalidResponse] = codes.Internal
	errorKindToGrpcStatusCode[InvalidRequest] = codes.InvalidArgument
	errorKindToGrpcStatusCode[InvalidResponseFormat] = codes.Internal
	errorKindToGrpcStatusCode[InvalidRequestFormat] = codes.Internal
	errorKindToGrpcStatusCode[Internal] = codes.Internal
	errorKindToGrpcStatusCode[InvalidJSONBody] = codes.Internal
	errorKindToGrpcStatusCode[InvalidHttpClient] = codes.Internal
	errorKindToGrpcStatusCode[PaginationFailed] = codes.Internal
}

type InternalError struct {
	err       error
	errorKind ErrorKind
	message   string
}

func NewInternalError(err error, errorKind ErrorKind, message string) *InternalError {
	return &InternalError{
		err:       err,
		errorKind: errorKind,
		message:   message,
	}
}

func (i *InternalError) RootError() error {
	return i.err
}

func (i *InternalError) Message() string {
	return i.message
}

func (i *InternalError) GrpcStatus() codes.Code {
	return i.errorKind.ToGrpcStatus()
}

func (i *InternalError) Error() string {
	return i.err.Error()
}
