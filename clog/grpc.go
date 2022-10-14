package clog

import (
	"bytes"
	"context"
	"github.com/golang/protobuf/jsonpb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"path"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

var (
	// Marshaller of Protobuf to JSON
	Marshaller = &jsonpb.Marshaler{}
	// TimestampLog call start.
	TimestampLog = true
	// ServiceField key.
	ServiceField = "service"
	// ServiceLog gRPC service name.
	ServiceLog = true
	// MethodField key.
	MethodField = "method"
	// MethodLog gRPC method name.
	MethodLog = true
	// DurationField key.
	DurationField = "dur"
	// DurationLog gRPC call duration.
	DurationLog = true
	// IPField key.
	IPField = "ip"
	// IPLog gRPC client IP.
	IPLog = true
	// MetadataField key.
	MetadataField = "md"
	// MetadataLog gRPC call metadata.
	MetadataLog = true
	// UserAgentField key.
	UserAgentField = "ua"
	// UserAgentLog gRPC client User Agent.
	UserAgentLog = true
	// ReqField key.
	ReqField = "req"
	// ReqLog gRPC request body.
	ReqLog = true
	// RespField key.
	RespField = "resp"
	// RespLog gRPC response body.
	RespLog = true
	// MaxSize to log gRPC bodies.
	MaxSize = 2048000
	// CodeField gRPC status code response.
	CodeField = "code"
	// MsgField gRPC response message.
	MsgField = "msg"
	// DetailsField gRPC response errors.
	DetailsField = "details"
	// UnaryMessageDefault of logging messages from unary.
	UnaryMessageDefault = "unary"
	// TraceIDField
	TraceIDField = "traceID"
)

// LogIncomingCall of gRPC method.
//
//	{
//		ServiceField: ExampleService,
//		MethodField: ExampleMethod,
//		DurationField: 1.00,
//	}
func LogIncomingCall(ctx context.Context, logger *zerolog.Event, method string, t time.Time, req interface{}) {
	LogTimestamp(logger, t)
	LogService(logger, method)
	LogMethod(logger, method)
	LogDuration(logger, t)
	LogRequest(logger, req)
	LogIncomingMetadata(ctx, logger)
}

func SetToContext(method string) *zerolog.Logger {
	logger := WithField(map[string]interface{}{
		ServiceField: path.Dir(method)[1:],
		MethodField:  path.Base(method),
		TraceIDField: generator(),
	})

	return logger
}

func LogIncomingRequest(ctx context.Context, logger *zerolog.Logger, method string, t time.Time, req interface{}) {
	log := logger.Info()
	LogTimestamp(log, t)
	LogService(log, method)
	LogMethod(log, method)
	LogDuration(log, t)
	LogRequest(log, req)
	LogIncomingMetadata(ctx, log)
	log.Send()
}

// LogTimestamp of call.
//
//	{
//		TimestampField: Timestamp,
//	}
func LogTimestamp(logger *zerolog.Event, t time.Time) {
	if TimestampLog {
		*logger = *logger.Time(zerolog.TimestampFieldName, t)
	}
}

// LogService of gRPC name.
//
//	{
//		ServiceField: gRPCServiceName,
//	}
func LogService(logger *zerolog.Event, method string) {
	if ServiceLog {
		*logger = *logger.Str(ServiceField, path.Dir(method)[1:])
	}
}

// LogMethod of gRPC call.
//
//	{
//		MethodField: gRPCMethodName,
//	}
func LogMethod(logger *zerolog.Event, method string) {
	if MethodLog {
		*logger = *logger.Str(MethodField, path.Base(method))
	}
}

// LogDuration in seconds of gRPC call.
//
//	{
//		DurationField: Timestamp,
//	}
func LogDuration(logger *zerolog.Event, t time.Time) {
	if DurationLog {
		*logger = *logger.Dur(DurationField, time.Since(t))
	}
}

// LogIP address of gRPC client, if assigned.
//
//	{
//		IpField: 127.0.0.1
//	}
func LogIP(ctx context.Context, logger *zerolog.Event) {
	if IPLog {
		if p, ok := peer.FromContext(ctx); ok {
			*logger = *logger.Str(IPField, p.Addr.String())
		}
	}
}

// LogRequest in JSON of gRPC Call, given Request is smaller than MaxSize (Default=2MB).
//
//	{
//		ReqField: {}
//	}
func LogRequest(e *zerolog.Event, req interface{}) {
	if ReqLog {
		if b := GetRawJSON(req); b != nil {
			*e = *e.RawJSON(ReqField, b.Bytes())
		}
	}
}

// LogResponse in JSON of gRPC Call, given Response is smaller than MaxSize (Default=2MB).
//
//	{
//		RespField: {}
//	}
func LogResponse(e *zerolog.Event, resp interface{}) {
	if RespLog {
		if b := GetRawJSON(resp); b != nil {
			*e = *e.RawJSON(RespField, b.Bytes())
		}
	}
}

// GetRawJSON converts a Protobuf message to JSON bytes if less than MaxSize.
func GetRawJSON(i interface{}) *bytes.Buffer {
	if pb, ok := i.(proto.Message); ok {
		b := &bytes.Buffer{}
		if err := Marshaller.Marshal(b, pb); err == nil && b.Len() < MaxSize {
			return b
		}
	}
	return nil
}

// LogIncomingMetadata or UserAgent field of incoming gRPC Request, if assigned.
//
//	{
//		MetadataField: {
//			MetadataKey1: MetadataValue1,
//		}
//	}
//
//	{
//		UserAgentField: "Client-assigned User-Agent",
//	}
func LogIncomingMetadata(ctx context.Context, e *zerolog.Event) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if MetadataLog {
			*e = *e.Dict(MetadataField, LogMetadata(&md))
			return
		} else if UserAgentLog {
			LogUserAgent(e, &md)
		}
	}
}

// LogMetadata of gRPC Request
//
//	{
//		MetadataField: {
//			MetadataKey1: MetadataValue1,
//		}
//	}
func LogMetadata(md *metadata.MD) *zerolog.Event {
	dict := zerolog.Dict()
	for i := range *md {
		dict = dict.Str(i, strings.Join(md.Get(i), ","))
	}
	return dict
}

// LogUserAgent of gRPC Client, if assigned.
//
//	{
//		UserAgentField: "Client-assigned User-Agent",
//	}
func LogUserAgent(logger *zerolog.Event, md *metadata.MD) {
	if ua := strings.Join(md.Get("user-agent"), ""); ua != "" {
		*logger = *logger.Str(UserAgentField, ua)
	}
}

// LogStatusError of gRPC Error Response.
//
//	{
//		Err: "An unexpected error occurred",
//		CodeField: "Unknown",
//		MsgField: "Error message returned from the server",
//		DetailsField: [Errors],
//	}
func LogStatusError(logger *zerolog.Event, err error) {
	statusErr := status.Convert(err)
	*logger = *logger.Err(err).Str(CodeField, statusErr.Code().String()).Str(MsgField, statusErr.Message()).Interface(DetailsField, statusErr.Details())
}

func UnaryServerInterceptorWithLogger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		now := time.Now()

		//grpc.SetTrailer(ctx, metadata.New(map[string]string{"my-key": "my-value2"}))
		//grpc.SendHeader(ctx, metadata.New(map[string]string{"my-key": "my-value1"}))
		log := SetToContext(info.FullMethod)
		ctx = context.WithValue(ctx, CLoggerKey, log)
		LogIncomingRequest(ctx, log, info.FullMethod, now, req)

		resp, err := handler(ctx, req)
		if log.Error().Enabled() {
			if err != nil {
				logger := log.Error()
				LogStatusError(logger, err)
				logger.Send()
			} else if log.Info().Enabled() {
				logger := log.Info()
				LogResponse(logger, resp)
				logger.Send()
			}
		}
		return resp, err
	}
}
