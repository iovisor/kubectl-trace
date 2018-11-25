package meta

import "testing"

func TestIsObjectName(t *testing.T) {
	tests := []struct {
		name       string
		objectName string
		want       bool
	}{
		{
			name:       "string with the right prefix",
			objectName: "kubectl-trace-1bb3ae39-efe8-11e8-9f29-8c164500a77e",
			want:       true,
		},
		{
			name:       "string with another prefix",
			objectName: "ekubectl-trace-1bb3ae39-efe8-11e8-9f29-8c164500a77e",
			want:       false,
		},
		{
			name:       "just an uuid",
			objectName: "1bb3ae39-efe8-11e8-9f29-8c164500a77e",
			want:       false,
		},
		{
			name:       "empty string",
			objectName: "",
			want:       false,
		},
		{
			name:       "just the prefix",
			objectName: "kubectl-trace",
			want:       false,
		},
		{
			name:       "just the prefix and a dash",
			objectName: "kubectl-trace-",
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsObjectName(tt.objectName); got != tt.want {
				t.Errorf("IsObjectName() = %v, want %v", got, tt.want)
			}
		})
	}
}
