package v1alpha1

import (
	"fmt"
	"strings"
)

// ValidateEngineAndImage checks spec.engine and spec.image consistency. Call on create/update
// before the controller applies DefaultSpec defaults. When spec.image is empty, image rules
// are skipped (the operator will default the image).
func (r *DistributedRedisCluster) ValidateEngineAndImage() error {
	if err := validateDatabaseEngineValue(r.Spec.Engine); err != nil {
		return err
	}
	if r.Spec.Image == "" {
		return nil
	}
	return validateImageMatchesEngine(r.EffectiveEngine(), r.Spec.Image)
}

func validateDatabaseEngineValue(e DatabaseEngine) error {
	switch e {
	case "", DatabaseEngineRedis, DatabaseEngineValkey:
		return nil
	default:
		return fmt.Errorf("spec.engine must be empty, %q, or %q, got %q", DatabaseEngineRedis, DatabaseEngineValkey, e)
	}
}

func validateImageMatchesEngine(engine DatabaseEngine, image string) error {
	lower := strings.ToLower(image)
	switch engine {
	case DatabaseEngineValkey:
		if !strings.Contains(lower, "valkey") {
			return fmt.Errorf("spec.image %q must contain %q when spec.engine is %q", image, "valkey", DatabaseEngineValkey)
		}
	default:
		if !strings.Contains(lower, "redis") {
			return fmt.Errorf("spec.image %q must contain %q when spec.engine is redis or empty", image, "redis")
		}
	}
	return nil
}
