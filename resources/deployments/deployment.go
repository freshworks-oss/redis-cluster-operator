package deployments

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

const (
	configMapMountPath = "/config"
)

// NewDeploymentForCR creates a new Deployment for the given Cluster.
func NewDeploymentForCR(instance *redisv1alpha1.RedisClusterBackup, cluster *redisv1alpha1.DistributedRedisCluster, dpName, secretName string) *appsv1.Deployment {
	deployment := generateDeploymentSpec(instance, cluster, dpName, secretName)
	return deployment
}

func generateDeploymentSpec(instance *redisv1alpha1.RedisClusterBackup, cluster *redisv1alpha1.DistributedRedisCluster, dpName, secretName string) *appsv1.Deployment {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      dpName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: generatePodTemplate(instance, cluster, secretName),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": instance.Spec.RedisClusterName,
				},
			},
		},
	}
	return deployment
}

func generatePodTemplate(instance *redisv1alpha1.RedisClusterBackup, cluster *redisv1alpha1.DistributedRedisCluster, secretName string) corev1.PodTemplateSpec {
	TerminationGracePeriodSeconds := int64(300)
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": instance.Spec.RedisClusterName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "rediscluster-util-container",
					Image: instance.Spec.Image,
					Env: []corev1.EnvVar{
						{
							Name: "REDIS_USER", // Change this to the desired environment variable name
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: secretName,
									},
									Key: "username",
								},
							},
						},
						{
							Name: "REDIS_PASSWORD", // Change this to the desired environment variable name
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: secretName,
									},
									Key: "password",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "configmap-volume", // Name of the ConfigMap volume
							MountPath: configMapMountPath, // Mount path for the ConfigMap
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				generateConfigmapVolumeSpec(instance), // Add the volume for the ConfigMap
			},
			TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
		},
	}
	return podTemplate
}

func generateConfigmapVolumeSpec(instance *redisv1alpha1.RedisClusterBackup) corev1.Volume {
	executeMode := int32(0755)
	configMapVolume := corev1.Volume{
		Name: "configmap-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: instance.Spec.UtilConfig,
				},
				DefaultMode: &executeMode,
			},
		},
	}
	return configMapVolume
}
