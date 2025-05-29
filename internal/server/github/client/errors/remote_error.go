package errors

import "google.golang.org/grpc/codes"

var httpToGRPCStatusCode = make(map[int]codes.Code)

func initHttpToGRPCStatusCodeMap() {
	httpToGRPCStatusCode[400] = codes.InvalidArgument
	httpToGRPCStatusCode[401] = codes.Unauthenticated
	httpToGRPCStatusCode[403] = codes.ResourceExhausted
	httpToGRPCStatusCode[404] = codes.NotFound
	httpToGRPCStatusCode[405] = codes.Unimplemented
	httpToGRPCStatusCode[406] = codes.ResourceExhausted
	httpToGRPCStatusCode[408] = codes.DeadlineExceeded
	httpToGRPCStatusCode[422] = codes.InvalidArgument
	httpToGRPCStatusCode[429] = codes.ResourceExhausted
	httpToGRPCStatusCode[500] = codes.Internal
	httpToGRPCStatusCode[501] = codes.Unimplemented
	httpToGRPCStatusCode[504] = codes.DeadlineExceeded
}

func init() {
	initHttpToGRPCStatusCodeMap()
}

type RemoteRequestError struct {
	err         error
	HttpMessage string
	// map http.StatusCode, set it to -1 if it is not an http error
	HttpStatusCode httpStatusCode
}

type httpStatusCode int

func (h httpStatusCode) toGRPCStatus() codes.Code {
	return httpToGRPCStatusCode[int(h)]
}

func NewRemoteError(err error, httpStatus int, responseBody string) *RemoteRequestError {
	return &RemoteRequestError{
		err:            err,
		HttpMessage:    responseBody,
		HttpStatusCode: httpStatusCode(httpStatus),
	}
}

func (r *RemoteRequestError) RootError() error {
	return r.err
}

func (r *RemoteRequestError) Message() string {
	return r.HttpMessage
}

func (r *RemoteRequestError) GrpcStatus() codes.Code {
	return r.HttpStatusCode.toGRPCStatus()
}

func (r *RemoteRequestError) Error() string {
	return r.err.Error()
}
