package goalx_test

import (
	"errors"
	"testing"

	goalx "github.com/vonbai/goalx"
)

type ConfigLayers struct {
	Base goalx.Config
}

type ResolveRequest struct {
	Preset  string
	Mode    goalx.Mode
	Source  string
	Comment string
}

type ResolvedConfig struct {
	Preset string
	Config goalx.Config
}

var errResolverNotImplemented = errors.New("resolver not implemented")

func resolveConfigFixture(layers ConfigLayers, req ResolveRequest) (ResolvedConfig, error) {
	_ = layers
	_ = req
	return ResolvedConfig{}, errResolverNotImplemented
}

func TestResolveConfigSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		layers ConfigLayers
		req    ResolveRequest
		want   ResolvedConfig
	}{
		{
			name: "explicit codex preset stays codex even with both engines present",
			layers: ConfigLayers{
				Base: goalx.Config{
					Name:      "demo",
					Mode:      goalx.ModeDevelop,
					Objective: "lock config state",
					Target:    goalx.TargetConfig{Files: []string{"README.md"}},
					Harness:   goalx.HarnessConfig{Command: "go test ./..."},
				},
			},
			req: ResolveRequest{
				Preset: "codex",
				Mode:   goalx.ModeDevelop,
			},
			want: ResolvedConfig{
				Preset: "codex",
				Config: goalx.Config{
					Preset: "codex",
					Mode:   goalx.ModeDevelop,
					Master: goalx.MasterConfig{Engine: "codex", Model: "gpt-5.4"},
				},
			},
		},
		{
			name: "unset preset auto-detects the best installed preset",
			layers: ConfigLayers{
				Base: goalx.Config{
					Name:      "demo",
					Mode:      goalx.ModeDevelop,
					Objective: "lock config state",
					Target:    goalx.TargetConfig{Files: []string{"README.md"}},
					Harness:   goalx.HarnessConfig{Command: "go test ./..."},
				},
			},
			req: ResolveRequest{
				Mode: goalx.ModeDevelop,
			},
			want: ResolvedConfig{
				Preset: "hybrid",
				Config: goalx.Config{
					Preset: "hybrid",
					Mode:   goalx.ModeDevelop,
					Master: goalx.MasterConfig{Engine: "claude-code", Model: "opus"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveConfigFixture(tt.layers, tt.req)
			if err != nil {
				t.Fatalf("resolveConfigFixture: %v", err)
			}
			if got.Preset != tt.want.Preset {
				t.Fatalf("preset = %q, want %q", got.Preset, tt.want.Preset)
			}
			if got.Config.Preset != tt.want.Config.Preset {
				t.Fatalf("config.preset = %q, want %q", got.Config.Preset, tt.want.Config.Preset)
			}
			if got.Config.Master.Engine != tt.want.Config.Master.Engine || got.Config.Master.Model != tt.want.Config.Master.Model {
				t.Fatalf("master = %#v, want %#v", got.Config.Master, tt.want.Config.Master)
			}
		})
	}
}
