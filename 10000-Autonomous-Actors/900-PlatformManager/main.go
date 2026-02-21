package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"

	"connectrpc.com/connect"
	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	whisper "Olympus2/90000-Enablement-Labs/P0000-pkg/000-whisper"
	platformv1 "OlympusGCP-Platform/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/platform/v1"
	"OlympusGCP-Platform/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/platform/v1/platformv1connect"
)

type PlatformServer struct {
	logger *whisper.WhisperLog
}

func (s *PlatformServer) PushImage(ctx context.Context, req *connect.Request[platformv1.PushRequest]) (*connect.Response[platformv1.StatusResponse], error) {
	slog.Info("Platform: Artifact Push", "image", req.Msg.Image)
	cmd := exec.Command("podman", "load")
	cmd.Stdin = bytes.NewReader(req.Msg.Data)
	cmd.Run()
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

func (s *PlatformServer) ListImages(ctx context.Context, req *connect.Request[platformv1.ListRequest]) (*connect.Response[platformv1.ListResponse], error) {
	output, _ := exec.Command("podman", "images", "--format", "{{.Repository}}").Output()
	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	var names []string
	for _, l := range lines {
		names = append(names, string(l))
	}
	return connect.NewResponse(&platformv1.ListResponse{Items: names}), nil
}

func (s *PlatformServer) WriteLog(ctx context.Context, req *connect.Request[platformv1.LogRequest]) (*connect.Response[platformv1.StatusResponse], error) {
	slog.Info("Platform: Cloud Log", "name", req.Msg.LogName, "msg", req.Msg.Message)
	s.logger.Log("LOG", "SUCCESS", req.Msg.LogName, req.Msg.Message, 0)
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

func (s *PlatformServer) RecordMetric(ctx context.Context, req *connect.Request[platformv1.MetricRequest]) (*connect.Response[platformv1.StatusResponse], error) {
	slog.Info("Platform: Cloud Metric", "name", req.Msg.Name, "val", req.Msg.Value)
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

func (s *PlatformServer) ValidateSpec(ctx context.Context, req *connect.Request[platformv1.SpecRequest]) (*connect.Response[platformv1.StatusResponse], error) {
	slog.Info("Platform: API Gateway Spec Validation")
	
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(req.Msg.OpenapiJson))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid OpenAPI spec: %v", err))
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("OpenAPI validation failed: %v", err))
	}

	slog.Info("Platform: OpenAPI Spec Validated Successfully", "title", doc.Info.Title, "version", doc.Info.Version)
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

func main() {
	slog.Info("PlatformManager: Booting Platform & Delivery Substrate...")
	w := whisper.New("PlatformManager", "gcp_platform.lpsv")
	defer w.Close()

	server := &PlatformServer{logger: w}
	mux := http.NewServeMux()
	mux.Handle(platformv1connect.NewPlatformServiceHandler(server))

	port := "8097"
	slog.Info("PlatformManager: Listening...", "addr", "localhost:"+port)
	http.ListenAndServe("localhost:"+port, h2c.NewHandler(mux, &http2.Server{}))
}
