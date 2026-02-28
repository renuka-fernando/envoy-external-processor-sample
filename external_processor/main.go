package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	ext_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	pb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

var _ ext_proc_v3.ExternalProcessorServer = &server{}

type server struct {
}

// Process implements ext_procv3.ExternalProcessorServer.
func (s *server) Process(processServer ext_proc_v3.ExternalProcessor_ProcessServer) error {
	ctx := processServer.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := processServer.Recv()
		if err == io.EOF {
			logrus.Debug("EOF")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}
		logrus.Info(fmt.Sprintf("******** Received Ext Processing Request ********\n%v", req))

		resp := &pb.ProcessingResponse{}
		switch value := req.Request.(type) {
		case *pb.ProcessingRequest_RequestHeaders:
			headers := value.RequestHeaders.Headers.GetHeaders()
			headersMap := make(map[string]string)
			for _, v := range headers {
				headersMap[v.Key] = string(v.GetRawValue())
			}

			httpMethod := headersMap[":method"]
			requestPath := headersMap[":path"]
			logrus.Print(fmt.Sprintf("******** Processing Request Headers ******** Method:%s, Path:%s", httpMethod, requestPath))

			// Check if dynamic metadata from the first ext_proc filter is present
			isSecondExtProc := false
			if metadataContext := req.GetMetadataContext(); metadataContext != nil {
				if ns, ok := metadataContext.GetFilterMetadata()["api_platform.policy_engine.envoy.filters.http.ext_proc"]; ok {
					if _, ok := ns.GetFields()["execute-the-path-rewrite"]; ok {
						isSecondExtProc = true
					}
				}
			}

			setHeaders := make([]*core_v3.HeaderValueOption, 0)
			if !isSecondExtProc {
				logrus.Print("******** First ext_proc: Setting x-target-cluster: upstream-service-2 ********")
				setHeaders = append(setHeaders, &core_v3.HeaderValueOption{
					Header: &core_v3.HeaderValue{
						Key:      "x-target-cluster",
						RawValue: []byte("upstream-service-2"),
					},
				})
			} else {
				logrus.Print("******** Second ext_proc: Setting :path: /hello-world-ext-proc ********")
				setHeaders = append(setHeaders, &core_v3.HeaderValueOption{
					Header: &core_v3.HeaderValue{
						Key:      ":path",
						RawValue: []byte("/hello-world-ext-proc"),
					},
				})
			}

			resp = &pb.ProcessingResponse{
				Response: &pb.ProcessingResponse_RequestHeaders{
					RequestHeaders: &pb.HeadersResponse{
						Response: &pb.CommonResponse{
							HeaderMutation: &pb.HeaderMutation{
								SetHeaders: setHeaders,
							},
						},
					},
				},
				DynamicMetadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"api_platform.policy_engine.envoy.filters.http.ext_proc": {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"execute-the-path-rewrite": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: true,
											},
										},
									},
								},
							},
						},
					},
				},
			}
		case *pb.ProcessingRequest_RequestBody:
			logrus.Print("******** Processing Request Body ******** body: ", string(value.RequestBody.Body))
			resp = &pb.ProcessingResponse{
				Response: &pb.ProcessingResponse_RequestBody{},
			}
		case *pb.ProcessingRequest_ResponseHeaders:
			headers := value.ResponseHeaders.Headers.GetHeaders()
			headersMap := make(map[string]string)
			for _, v := range headers {
				headersMap[v.Key] = string(v.GetRawValue())
			}

			status := headersMap[":status"]
			logrus.Print(fmt.Sprintf("******** Processing Response Headers ******** status:%v", status))
			resp = &pb.ProcessingResponse{
				Response: &pb.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: &pb.HeadersResponse{
						Response: &pb.CommonResponse{
							HeaderMutation: &pb.HeaderMutation{
								SetHeaders: []*core_v3.HeaderValueOption{
									{
										Header: &core_v3.HeaderValue{
											Key:      "hello",
											RawValue: []byte("world"),
										},
									},
									{
										Header: &core_v3.HeaderValue{
											Key:      "test",
											RawValue: []byte("renuka"),
										},
									},
									// {
									// 	Header: &core_v3.HeaderValue{
									// 		Key:      "Content-Length",
									// 		RawValue: []byte(fmt.Sprint(len("Hello World"))),
									// 	},
									// },
								},
							},
						},
					},
				},
				DynamicMetadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"api_platform.policy_engine.envoy.filters.http.ext_proc": {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"foo-resp": {
											Kind: &structpb.Value_StringValue{
												StringValue: "bar-resp",
											},
										},
										"foo": {
											Kind: &structpb.Value_StringValue{
												StringValue: "bar-resp2",
											},
										},
									},
								},
							},
						},
					},
				},
			}
		case *pb.ProcessingRequest_ResponseBody:
			logrus.Print("******** Processing Response Body ******** body: ", string(value.ResponseBody.Body))
			resp = &pb.ProcessingResponse{
				Response: &pb.ProcessingResponse_ResponseBody{
					ResponseBody: &pb.BodyResponse{
						Response: &pb.CommonResponse{
							BodyMutation: &pb.BodyMutation{
								Mutation: &pb.BodyMutation_Body{
									Body: []byte("Hello World"),
								},
							},
						},
					},
				},
			}
		default:
			logrus.Debug(fmt.Sprintf("Unknown Request type %v\n", value))
		}
		if err := processServer.Send(resp); err != nil {
			logrus.Debug(fmt.Sprintf("send error %v", err))
		}
	}
}

func main() {
	port := flag.Int("port", 9001, "gRPC port")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen to %d: %v", *port, err)
	}

	gs := grpc.NewServer()
	ext_proc_v3.RegisterExternalProcessorServer(gs, &server{})
	log.Printf("starting gRPC server on: %d\n", *port)
	gs.Serve(lis)
}
