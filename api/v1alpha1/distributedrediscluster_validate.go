package v1alpha1

import (
	"fmt"
	"strconv"
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

// ValidateBootstrapNode validates spec.bootstrapNode when present.
func (r *DistributedRedisCluster) ValidateBootstrapNode() error {
	if r.Spec.BootstrapNode == nil {
		return nil
	}
	bootstrap := r.Spec.BootstrapNode
	host := strings.TrimSpace(bootstrap.Host)
	if host == "" {
		return fmt.Errorf("spec.bootstrapNode.host is required when bootstrapNode is set")
	}
	if bootstrap.Port != "" {
		if err := validateBootstrapPort(bootstrap.Port); err != nil {
			return fmt.Errorf("spec.bootstrapNode.port: %w", err)
		}
	}
	if r.IsRestoreFromBackup() {
		return fmt.Errorf("spec.bootstrapNode cannot be set when restore from backup is configured")
	}
	return nil
}

func validateBootstrapPort(port string) error {
	port = strings.TrimSpace(port)
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return fmt.Errorf("must be a numeric port, got %q", port)
	}
	if p == 0 {
		return fmt.Errorf("must be between 1 and 65535, got %q", port)
	}
	return nil
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
