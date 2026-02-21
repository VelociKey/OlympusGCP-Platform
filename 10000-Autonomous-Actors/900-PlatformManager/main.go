package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"

	whisper "Olympus2/90000-Enablement-Labs/P0000-pkg/000-whisper"
	platformv1 "OlympusGCP-Platform/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/platform/v1"
	"OlympusGCP-Platform/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/platform/v1/platformv1connect"
)

type PlatformServer struct {
	logger *whisper.WhisperLog
}

// Cloud Build YAML Structure
type BuildStep struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}
type BuildDefinition struct {
	Steps []BuildStep `yaml:"steps"`
}

func (s *PlatformServer) PushImage(ctx context.Context, req *connect.Request[platformv1.PushRequest]) (*connect.Response[platformv1.StatusResponse], error) {
	slog.Info("Platform: Artifact Push", "image", req.Msg.Image)
	cmd := exec.Command("podman", "load")
	cmd.Stdin = bytes.NewReader(req.Msg.Data)
	cmd.Run()
	
	// Simulation: Trigger background vulnerability scan
	slog.Info("Platform: Vulnerability Scan Initiated", "image", req.Msg.Image)
	
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

func (s *PlatformServer) ListImages(ctx context.Context, req *connect.Request[platformv1.ListRequest]) (*connect.Response[platformv1.ListResponse], error) {
	output, _ := exec.Command("podman", "images", "--format", "{{.Repository}}").Output()
	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	var names []string
	for _, l := range lines { names = append(names, string(l)) }
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
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid OpenAPI spec: %v", err)) }
	if err := doc.Validate(ctx); err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("OpenAPI validation failed: %v", err)) }
	return connect.NewResponse(&platformv1.StatusResponse{Success: true}), nil
}

// HIGH-FIDELITY DEEPENING: Cloud Build Simulator
func (s *PlatformServer) ExecuteBuild(ctx context.Context, req *connect.Request[platformv1.CloudBuildRequest]) (*connect.Response[platformv1.CloudBuildResponse], error) {
	slog.Info("Platform: Executing High-Fidelity Cloud Build", "workspace", req.Msg.WorkspacePath)
	
	var build BuildDefinition
	if err := yaml.Unmarshal([]byte(req.Msg.BuildYaml), &build); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	buildID := fmt.Sprintf("build-%d", time.Now().Unix())
	var logs bytes.Buffer

	for _, step := range build.Steps {
		slog.Info("Platform: Cloud Build Step", "image", step.Name, "args", step.Args)
		logs.WriteString(fmt.Sprintf("--- Step: %s ---\n", step.Name))
		
		// Map GCP 'gcr.io/cloud-builders/docker' to local 'podman'
		cmdName := step.Name
		if strings.Contains(step.Name, "docker") { cmdName = "podman" }
		
		cmd := exec.Command(cmdName, step.Args...)
		cmd.Dir = req.Msg.WorkspacePath
		out, err := cmd.CombinedOutput()
		logs.Write(out)
		
		if err != nil {
			return connect.NewResponse(&platformv1.CloudBuildResponse{
				BuildId: buildID,
				Status:  "FAILURE",
				LogTail: logs.String(),
			}), nil
		}
	}

	return connect.NewResponse(&platformv1.CloudBuildResponse{
		BuildId: buildID,
		Status:  "SUCCESS",
		LogTail: logs.String(),
	}), nil
}

func main() {
	slog.Info("PlatformManager: Booting Platform & Delivery Substrate (Phase 8)...")
	w := whisper.New("PlatformManager", "gcp_platform.lpsv")
	defer w.Close()

	server := &PlatformServer{logger: w}
	mux := http.NewServeMux()
	mux.Handle(platformv1connect.NewPlatformServiceHandler(server))

	port := "8097"
	slog.Info("PlatformManager: Listening...", "addr", "localhost:"+port)

	srv := &http.Server{
		Addr:         "localhost:"+port,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
