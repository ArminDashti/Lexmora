package service

import "testing"

func TestOpenCodeGoEndpointForModel(t *testing.T) {
	cases := []struct {
		model string
		kind  opencodeGoEndpointKind
	}{
		{"qwen3.8-flash", opencodeGoMessages},
		{"opencode-go/qwen3.8-flash", opencodeGoMessages},
		{"gpt-6-luna", opencodeGoResponses},
		{"longcat-2.5-preview-free", opencodeGoChat},
	}
	for _, tc := range cases {
		kind, err := opencodeGoEndpointForModel(tc.model)
		if err != nil {
			t.Fatalf("model %q: %v", tc.model, err)
		}
		if kind != tc.kind {
			t.Fatalf("model %q: got %q want %q", tc.model, kind, tc.kind)
		}
	}

	_, err := opencodeGoEndpointForModel("not-a-real-model")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}
