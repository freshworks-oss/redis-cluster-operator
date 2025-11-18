package osm

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	awsconst "kmodules.xyz/constants/aws"
	azconst "kmodules.xyz/constants/azure"
	googconst "kmodules.xyz/constants/google"
	osconst "kmodules.xyz/constants/openstack"

	"github.com/appscode/go/types"
	otx "github.com/appscode/osm/context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"gomodules.xyz/stow"
	"gomodules.xyz/stow/azure"
	gcs "gomodules.xyz/stow/google"
	"gomodules.xyz/stow/local"
	stows3 "gomodules.xyz/stow/s3"
	"gomodules.xyz/stow/swift"
	core "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	api "kmodules.xyz/objectstore-api/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SecretMountPath = "/etc/rclone"
	CaCertFileName  = "ca.crt"
)

func CheckBucketAccess(client client.Client, spec api.Backend, namespace string) error {
	cfg, err := NewOSMContext(client, spec, namespace)
	if err != nil {
		return err
	}
	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return err
	}
	bucket, err := spec.Container()
	if err != nil {
		return err
	}
	c, err := loc.Container(bucket)
	if err != nil {
		return err
	}
	return c.HasWriteAccess()
}

func NewOSMContext(client client.Client, spec api.Backend, namespace string) (*otx.Context, error) {
	secretConfig := make(map[string][]byte)

	if spec.StorageSecretName != "" {
		secret := &core.Secret{}
		err := client.Get(context.TODO(), ktypes.NamespacedName{
			Name:      spec.StorageSecretName,
			Namespace: namespace,
		}, secret)
		if err != nil {
			return nil, err
		}
		secretConfig = secret.Data
	}

	nc := &otx.Context{
		Name:   "objectstore",
		Config: stow.ConfigMap{},
	}

	if spec.S3 != nil {
		nc.Provider = stows3.Kind

		keyID, foundKeyID := secretConfig[awsconst.AWS_ACCESS_KEY_ID]
		key, foundKey := secretConfig[awsconst.AWS_SECRET_ACCESS_KEY]
		if foundKey && foundKeyID {
			nc.Config[stows3.ConfigAccessKeyID] = string(keyID)
			nc.Config[stows3.ConfigSecretKey] = string(key)
			nc.Config[stows3.ConfigAuthType] = "accesskey"
		} else {
			nc.Config[stows3.ConfigAuthType] = "iam"
		}
		if spec.S3.Endpoint == "" || strings.HasSuffix(spec.S3.Endpoint, ".amazonaws.com") {
			// Using s3 and not s3-compatible service like minio or rook, etc. Now, find region
			var cfg aws.Config
			var err error
			ctx := context.TODO()

			if nc.Config[stows3.ConfigAuthType] == "iam" {
				// Load default AWS configuration with region detection
				cfg, err = config.LoadDefaultConfig(ctx)
				if err != nil {
					// Fallback: try to get region from EC2 metadata
					imdsClient := imds.NewFromConfig(cfg)
					regionResp, regionErr := imdsClient.GetRegion(ctx, &imds.GetRegionInput{})
					if regionErr == nil {
						cfg.Region = regionResp.Region
					} else {
						// Last resort: use us-east-1
						cfg.Region = "us-east-1"
					}
				}
			} else {
				// Use static credentials
				cfg, err = config.LoadDefaultConfig(ctx,
					config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
						string(keyID), string(key), "",
					)),
					config.WithRegion("us-east-1"), // Default region
				)
			}
			if err != nil {
				return nil, err
			}

			// Create S3 client and get bucket location
			s3Client := s3.NewFromConfig(cfg)
			out, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
				Bucket: types.StringP(spec.S3.Bucket),
			})
			if err != nil {
				return nil, err
			}

			// Handle the bucket location constraint
			region := "us-east-1" // Default for us-east-1 (returns empty constraint)
			if out.LocationConstraint != "" {
				region = string(out.LocationConstraint)
			}
			nc.Config[stows3.ConfigRegion] = region
		} else {
			nc.Config[stows3.ConfigEndpoint] = spec.S3.Endpoint
			u, err := url.Parse(spec.S3.Endpoint)
			if err != nil {
				return nil, err
			}
			nc.Config[stows3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")

			cacertData, ok := secretConfig[awsconst.CA_CERT_DATA]
			if ok && u.Scheme == "https" {
				nc.Config[stows3.ConfigCACertData] = string(cacertData)
			}
		}
		return nc, nil
	} else if spec.GCS != nil {
		nc.Provider = gcs.Kind
		nc.Config[gcs.ConfigProjectId] = string(secretConfig[googconst.GOOGLE_PROJECT_ID])
		nc.Config[gcs.ConfigJSON] = string(secretConfig[googconst.GOOGLE_SERVICE_ACCOUNT_JSON_KEY])
		return nc, nil
	} else if spec.Azure != nil {
		nc.Provider = azure.Kind
		nc.Config[azure.ConfigAccount] = string(secretConfig[azconst.AZURE_ACCOUNT_NAME])
		nc.Config[azure.ConfigKey] = string(secretConfig[azconst.AZURE_ACCOUNT_KEY])
		return nc, nil
	} else if spec.Local != nil {
		nc.Provider = local.Kind
		nc.Config[local.ConfigKeyPath] = spec.Local.MountPath
		return nc, nil
	} else if spec.Swift != nil {
		nc.Provider = swift.Kind
		// https://github.com/restic/restic/blob/master/src/restic/backend/swift/config.go
		for _, val := range []struct {
			stowKey   string
			secretKey string
		}{
			// v2/v3 specific
			{swift.ConfigUsername, osconst.OS_USERNAME},
			{swift.ConfigKey, osconst.OS_PASSWORD},
			{swift.ConfigRegion, osconst.OS_REGION_NAME},
			{swift.ConfigTenantAuthURL, osconst.OS_AUTH_URL},

			// v3 specific
			{swift.ConfigDomain, osconst.OS_USER_DOMAIN_NAME},
			{swift.ConfigTenantName, osconst.OS_PROJECT_NAME},
			{swift.ConfigTenantDomain, osconst.OS_PROJECT_DOMAIN_NAME},

			// v2 specific
			{swift.ConfigTenantId, osconst.OS_TENANT_ID},
			{swift.ConfigTenantName, osconst.OS_TENANT_NAME},

			// v1 specific
			{swift.ConfigTenantAuthURL, osconst.ST_AUTH},
			{swift.ConfigUsername, osconst.ST_USER},
			{swift.ConfigKey, osconst.ST_KEY},

			// Manual authentication
			{swift.ConfigStorageURL, osconst.OS_STORAGE_URL},
			{swift.ConfigAuthToken, osconst.OS_AUTH_TOKEN},
		} {
			if _, exists := nc.Config.Config(val.stowKey); !exists {
				nc.Config[val.stowKey] = string(secretConfig[val.secretKey])
			}
		}
		return nc, nil
	}
	return nil, errors.New("no storage provider is configured")
}
