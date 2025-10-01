package configmaps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

const (
	RestoreSucceeded = "succeeded"

	RedisConfKey = "redis.conf"

	Aclfile = "users.acl"
)

var ErrBadInput = errors.New("admin-secret-missing")

// NewConfigMapForCR creates a new ConfigMap for the given Cluster
func NewConfigMapForCR(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) *corev1.ConfigMap {
	// Do CLUSTER FAILOVER when master down

	// Fixed Nodes.conf does not update IP address of a node when IP changes after restart,
	// see more https://github.com/antirez/redis/issues/4645.
	fixIPContent := `#!/bin/sh
echo "Delaying bootup for 5 seconds to honour the node timeout and seamless failover" >&2
sleep 5
CLUSTER_CONFIG="/data/nodes.conf"
if [ -f ${CLUSTER_CONFIG} ]; then
    if [ -z "${POD_IP}" ]; then
    echo "Unable to determine Pod IP address!"
    exit 1
    fi
    echo "Updating my IP to ${POD_IP} in ${CLUSTER_CONFIG}"
    sed -i.bak -e "/myself/ s/ .*:6379@16379/ ${POD_IP}:6379@16379/" ${CLUSTER_CONFIG}
fi
exec "$@"`

	redisConfContent := generateRedisConfContent(cluster.Spec.Config, cluster.Spec.Image)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            RedisConfigMapName(cluster.Name),
			Namespace:       cluster.Namespace,
			Labels:          labels,
			OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
		},
		Data: map[string]string{
			"fix-ip.sh":  fixIPContent,
			RedisConfKey: redisConfContent,
		},
	}
}

func NewACLConfigMap(conflog logr.Logger, client client.Client, cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) (*corev1.ConfigMap, map[string]string, error, bool) {
	//	secretversion := make(map[string]string)
	aclfileContent, secretversion, err := GenerateAclconf(conflog, client, cluster)
	if err != nil {
		if errors.Is(err, ErrBadInput) {
			return nil, secretversion, err, true
		} else {
			return nil, secretversion, err, false
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            AclConfigMapName(cluster.Name),
			Namespace:       cluster.Namespace,
			Labels:          labels,
			OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
		},
		Data: map[string]string{
			Aclfile: aclfileContent,
		},
	}, secretversion, nil, false

}

func Getsecretversions(conflog logr.Logger, client client.Client, cluster *redisv1alpha1.DistributedRedisCluster) (map[string]string, error) {

	secretversion := make(map[string]string)

	if cluster.Spec.AdminSecret == nil {

		err := fmt.Errorf("invalid custom resource - admin secret missing")
		conflog.Error(err, "admin secret missing")
		return secretversion, err
	} else {
		_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.AdminSecret.Name)
		if err != nil {
			conflog.Error(err, "admin secret unable to get")
			return secretversion, err

		} else {
			secretversion[cluster.Spec.AdminSecret.Name] = versionID
		}
	}

	if cluster.Spec.DefaultSecret == nil {
		conflog.Info("Getsecretversions", "defaultsecret", "default secret not given generating one. Ideally should not come here")
		secretversion["default-secret"] = "default"
	} else {
		_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.DefaultSecret.Name)
		if err != nil {
			conflog.Error(err, "default secret unable to get")
			return secretversion, err

		} else {

			secretversion[cluster.Spec.DefaultSecret.Name] = versionID
		}
	}

	if cluster.Spec.AdditionalSecret != nil {
		for _, ref := range cluster.Spec.AdditionalSecret {
			sceretrefname := ref.Name
			//conflog.Info("Getsecretversions", "generating acl for", sceretrefname)
			_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, sceretrefname)
			if err != nil {
				conflog.Error(err, "additional secret unable to get")
				return secretversion, err
			} else {

				secretversion[sceretrefname] = versionID
			}
		}
	}
	return secretversion, nil
}

/*
	func Getsecretversions(conflog logr.Logger, client client.Client, cluster *redisv1alpha1.DistributedRedisCluster) (map[string]string, error) {
		secretversion := make(map[string]string)

		if cluster.Spec.AdminSecret == nil {
			err := fmt.Errorf("admin secret refrence not defined in CR")
			conflog.Error(err, "in Getsecretversions:")
			secretversion["admin-secret"] = "default"
		} else {
			conflog.Info("Getsecretversions", "fetting secret revision for", cluster.Spec.AdminSecret.Name)
			_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.AdminSecret.Name)
			if err != nil {
				conflog.Error(err, "admin secret unable to get")
				return secretversion, err

			} else {
				secretversion[cluster.Spec.AdminSecret.Name] = versionID
			}
		}

		if cluster.Spec.DefaultSecret == nil {
			err := fmt.Errorf("default secret refrence not defined in CR")
			conflog.Error(err, "in Getsecretversions:")
			return secretversion, err
		} else {
			conflog.Info("Getsecretversions", "fetting secret revision for", cluster.Spec.DefaultSecret.Name)
			_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.DefaultSecret.Name)
			if err != nil {
				conflog.Error(err, "default secret unable to get")
				return secretversion, err

			} else {
				secretversion[cluster.Spec.DefaultSecret.Name] = versionID
			}
		}

		if cluster.Spec.AdditionalSecret != nil {
			for _, ref := range cluster.Spec.AdditionalSecret {
				sceretrefname := ref.Name
				conflog.Info("Getsecretversions", "fetting secret revision for", sceretrefname)
				_, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, sceretrefname)
				if err != nil {
					conflog.Error(err, "additional secret unable to get")
				} else {
					secretversion[sceretrefname] = versionID
				}
			}
		}
		return secretversion, nil
	}
*/
func GenerateAclconf(conflog logr.Logger, client client.Client, cluster *redisv1alpha1.DistributedRedisCluster) (string, map[string]string, error) {
	var buffer bytes.Buffer

	secretversion := make(map[string]string)

	if cluster.Spec.AdminSecret == nil {

		err := fmt.Errorf("invalid custom resource: %w", ErrBadInput)
		conflog.Error(err, "admin secret missing")
		return "", secretversion, err
	} else {
		acldata, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.AdminSecret.Name)
		if err != nil {
			conflog.Error(err, "admin secret unable to create")
			return "", secretversion, err

		} else {
			buffer.WriteString(acldata)
			buffer.WriteString("\n")
			secretversion[cluster.Spec.AdminSecret.Name] = versionID
		}
	}

	if cluster.Spec.DefaultSecret == nil {
		conflog.Info("generateAclconf", "defaultsecret", "default secret not given generating one. Ideally should not come here")
		buffer.WriteString("user default on ~* +@all -@dangerous >defaultpwd")
		buffer.WriteString("\n")
		secretversion["default-secret"] = "default"
	} else {
		acldata, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, cluster.Spec.DefaultSecret.Name)
		if err != nil {
			conflog.Error(err, "default secret unable to create")
			return "", secretversion, err

		} else {
			buffer.WriteString(acldata)
			buffer.WriteString("\n")
			secretversion[cluster.Spec.DefaultSecret.Name] = versionID
		}
	}

	if cluster.Spec.AdditionalSecret != nil {
		for _, ref := range cluster.Spec.AdditionalSecret {
			sceretrefname := ref.Name
			//conflog.Info("generateAclconf", "generating acl for", sceretrefname)
			acldata, versionID, err := GetAclStringFromSecrets(conflog, client, cluster, sceretrefname)
			if err != nil {
				conflog.Error(err, "additional secret unable to create")
				return "", secretversion, err
			} else {
				buffer.WriteString(acldata)
				buffer.WriteString("\n")
				secretversion[sceretrefname] = versionID
			}
		}
	}
	return buffer.String(), secretversion, nil
}
func generateRedisConfContent(configMap map[string]string, redisImage string) string {

	var buffer bytes.Buffer

	// Check if Redis version supports ACL (6.0+)
	if supportsACL(redisImage) {
		// Write the acl file
		aclfilepath := "/acl/" + Aclfile
		buffer.WriteString(fmt.Sprintf("%s %s", "aclfile", aclfilepath))
		buffer.WriteString("\n")
	}

	buffer.WriteString(fmt.Sprintf("%s %s", "dir", "/backup"))
	buffer.WriteString("\n")

	if configMap == nil {
		return buffer.String()
	}

	keys := make([]string, 0, len(configMap))
	for k := range configMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := configMap[k]
		if len(v) == 0 {
			continue
		}
		buffer.WriteString(fmt.Sprintf("%s %s", k, v))
		buffer.WriteString("\n")
	}
	return buffer.String()
}

// supportsACL checks if the Redis version supports ACL (Redis 6.0+)
func supportsACL(redisImage string) bool {
	// Simple version detection based on image tag
	// This is a basic implementation - in production, you might want more sophisticated version parsing
	if strings.Contains(redisImage, ":5.") || strings.Contains(redisImage, ":4.") || strings.Contains(redisImage, ":3.") {
		return false
	}
	// Default to true for Redis 6.0+ and unknown versions
	return true
}

func RedisConfigMapName(clusterName string) string {
	if strings.HasPrefix(clusterName, redisv1alpha1.CrResourcesprefix) {
		namespace := clusterName[len(redisv1alpha1.CrResourcesprefix):]
		return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, namespace, "redis-config")
	}
	return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, clusterName, "redis-config")
}

func NewConfigMapForRestore(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            RestoreConfigMapName(cluster.Name),
			Namespace:       cluster.Namespace,
			Labels:          labels,
			OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
		},
		Data: map[string]string{
			RestoreSucceeded: strconv.Itoa(0),
		},
	}
}

func RestoreConfigMapName(clusterName string) string {
	if strings.HasPrefix(clusterName, redisv1alpha1.CrResourcesprefix) {
		namespace := clusterName[len(redisv1alpha1.CrResourcesprefix):]
		return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, namespace, "restore")
	}
	return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, clusterName, "restore")
}

func AclConfigMapName(clusterName string) string {
	if strings.HasPrefix(clusterName, redisv1alpha1.CrResourcesprefix) {
		namespace := clusterName[len(redisv1alpha1.CrResourcesprefix):]
		return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, namespace, "redis-acl")
	}
	return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, clusterName, "redis-acl")
}

func RedisInitConfigMapName(clusterName string) string {
	if strings.HasPrefix(clusterName, redisv1alpha1.CrResourcesprefix) {
		namespace := clusterName[len(redisv1alpha1.CrResourcesprefix):]
		return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, namespace, "redis-init")
	}
	return fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, clusterName, "redis-init")
}

func GetAclStringFromSecrets(conflog logr.Logger, client client.Client, cluster *redisv1alpha1.DistributedRedisCluster, secretname string) (string, string, error) {

	aclsecret, versionId, err := GetAcldataFromSecrets(client, cluster, secretname)
	if err != nil {
		conflog.Error(err, "Unable to get ACL data from secrets")
		return "", "", err
	}
	aclString := fmt.Sprintf("user %s on %s #%s", aclsecret["username"], aclsecret["acl"], GetHashedPassword(aclsecret["password"]))
	//conflog.Info("GetAclStringFromSecrets", "aclstring", aclString)
	if _, ok := aclsecret["old_password"]; ok {
		aclString = aclString + fmt.Sprintf(" #%s", GetHashedPassword(aclsecret["old_password"]))
	}
	return aclString, versionId, nil
}

// Extract the auth,acldata from secrets.
func GetAcldataFromSecrets(client client.Client, cluster *redisv1alpha1.DistributedRedisCluster, secretName string) (map[string]string, string, error) {
	values := make(map[string]string)
	secret := &corev1.Secret{}
	// TODO : DR-1638:This is not the right place to do this operation. This should be done in the controller. To be fixed in cleanup - PRATHEEP
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      secretName,
		Namespace: cluster.Namespace,
	}, secret)
	if err != nil {
		return values, "", err
	}

	// Check required fields in the secret
	requiredFields := []string{"username", "password", "acl"}
	for _, field := range requiredFields {
		if data, ok := secret.Data[field]; !ok || len(data) == 0 {
			return nil, secret.ResourceVersion, fmt.Errorf("secret '%s' is missing data for '%s'", secretName, field)
		}
		values[field] = strings.TrimSpace(string(secret.Data[field]))
	}
	if data, ok := secret.Data["old_password"]; ok && len(data) != 0 {
		values["old_password"] = strings.TrimSpace(string(secret.Data["old_password"]))
	}
	return values, secret.ResourceVersion, nil
}

func GetHashedPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum(nil))
}
