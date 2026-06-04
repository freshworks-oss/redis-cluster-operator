package v1alpha1

import (
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEffectiveEngine(t *testing.T) {
	tests := []struct {
		name   string
		engine DatabaseEngine
		want   DatabaseEngine
	}{
		{"empty defaults redis", "", DatabaseEngineRedis},
		{"redis explicit", DatabaseEngineRedis, DatabaseEngineRedis},
		{"valkey", DatabaseEngineValkey, DatabaseEngineValkey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{Engine: tt.engine},
			}
			if got := in.EffectiveEngine(); got != tt.want {
				t.Errorf("EffectiveEngine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSpec_DefaultImage(t *testing.T) {
	log := logr.Discard()

	t.Run("empty engine gets redis default image", func(t *testing.T) {
		in := &DistributedRedisCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "c1"},
			Spec:       DistributedRedisClusterSpec{MasterSize: 3},
		}
		in.DefaultSpec(log)
		if in.Spec.Image != defaultRedisImage {
			t.Fatalf("image = %q, want %q", in.Spec.Image, defaultRedisImage)
		}
	})

	t.Run("valkey engine gets valkey default image", func(t *testing.T) {
		in := &DistributedRedisCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "c1"},
			Spec: DistributedRedisClusterSpec{
				MasterSize: 3,
				Engine:     DatabaseEngineValkey,
			},
		}
		in.DefaultSpec(log)
		if in.Spec.Image != defaultValkeyImage {
			t.Fatalf("image = %q, want %q", in.Spec.Image, defaultValkeyImage)
		}
	})

	t.Run("explicit image unchanged", func(t *testing.T) {
		in := &DistributedRedisCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "c1"},
			Spec: DistributedRedisClusterSpec{
				MasterSize: 3,
				Image:      "redis:custom",
			},
		}
		in.DefaultSpec(log)
		if in.Spec.Image != "redis:custom" {
			t.Fatalf("image = %q", in.Spec.Image)
		}
	})
}
