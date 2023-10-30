/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DistributedRedisClusterSpec defines the desired state of DistributedRedisCluster
type DistributedRedisClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of DistributedRedisCluster. Edit distributedrediscluster_types.go to remove/update
	InitContainers   []corev1.Container            `json:"initContainers,omitempty"`
	ExtraContainers  []corev1.Container            `json:"extraContainers,omitempty"`
	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Command          []string                      `json:"command,omitempty"`
	Env              []corev1.EnvVar               `json:"env,omitempty"`
	MasterSize       int32                         `json:"masterSize,omitempty"`
	ClusterReplicas  int32                         `json:"clusterReplicas,omitempty"`
	ServiceName      string                        `json:"serviceName,omitempty"`
	Config           map[string]string             `json:"config,omitempty"`
	// Set RequiredAntiAffinity to force the master-slave node anti-affinity.
	RequiredAntiAffinity     bool                          `json:"requiredAntiAffinity,omitempty"`
	Affinity                 *corev1.Affinity              `json:"affinity,omitempty"`
	NodeSelector             map[string]string             `json:"nodeSelector,omitempty"`
	ToleRations              []corev1.Toleration           `json:"toleRations,omitempty"`
	SecurityContext          *corev1.PodSecurityContext    `json:"securityContext,omitempty"`
	ContainerSecurityContext *corev1.SecurityContext       `json:"containerSecurityContext,omitempty"`
	Annotations              map[string]string             `json:"annotations,omitempty"`
	Storage                  *RedisStorage                 `json:"storage,omitempty"`
	Resources                *corev1.ResourceRequirements  `json:"resources,omitempty"`
	AdminSecret              *corev1.LocalObjectReference  `json:"adminUser,omitempty"`
	DefaultSecret            *corev1.LocalObjectReference  `json:"defaultUser,omitempty"`
	AdditionalSecret         []corev1.LocalObjectReference `json:"additionalUsers,omitempty"`
<<<<<<< Updated upstream
	UtilConfig               string                        `json:"utilConfig"`
	Monitor                  *AgentSpec                    `json:"monitor,omitempty"`
=======
>>>>>>> Stashed changes
	Restore                  *RestoreSpec                  `json:"restoreContainer,omitempty"`
	TerminationGracePeriod   *int64                        `json:"terminationGracePeriod,omitempty"`
	CustomLivenessProbe      corev1.Probe                  `json:"customLivenessProbe"`
	ReadinessProbe           corev1.Probe                  `json:"readinessProbe"`
}

type RestoreSpec struct {
	BackupSource *BackupSourceSpec `json:"backupSource,omitempty"`
}

type BackupSourceSpec struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// Arguments to the restore job
	Args []string `json:"args,omitempty"`
}

// RedisStorage defines the structure used to store the Redis Data
type RedisStorage struct {
	Size        resource.Quantity `json:"size"`
	Type        StorageType       `json:"type"`
	Class       string            `json:"class"`
	DeleteClaim bool              `json:"deleteClaim,omitempty"`
}

// DistributedRedisClusterStatus defines the observed state of DistributedRedisCluster
type DistributedRedisClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status               ClusterStatus      `json:"status"`
	SecretStatus         string             `json:"secretstatus"`
	SecretsVer           map[string]string  `json:"secretverisons"`
	Reason               string             `json:"reason,omitempty"`
	NumberOfMaster       int32              `json:"numberOfMaster,omitempty"`
	MinReplicationFactor int32              `json:"minReplicationFactor,omitempty"`
	MaxReplicationFactor int32              `json:"maxReplicationFactor,omitempty"`
	Nodes                []RedisClusterNode `json:"nodes"`
	// +optional
	Restore Restore `json:"restore,omitempty"`
}

type Restore struct {
	Phase RestorePhase `json:"phase,omitempty"`
	//Backup *RedisClusterBackup `json:"backup, omitempty"`
}

// RedisClusterNode represent a RedisCluster Node
type RedisClusterNode struct {
	ID          string    `json:"id"`
	Role        RedisRole `json:"role"`
	IP          string    `json:"ip"`
	Port        string    `json:"port"`
	Slots       []string  `json:"slots,omitempty"`
	MasterRef   string    `json:"masterRef,omitempty"`
	PodName     string    `json:"podName"`
	NodeName    string    `json:"nodeName"`
	StatefulSet string    `json:"statefulSet"`
}

// +genclient
// +kubebuilder:printcolumn:name="replicas",type=number,JSONPath=`.spec.clusterReplicas`
// +kubebuilder:printcolumn:name="service",type=string,JSONPath=`.spec.serviceName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced, shortName={drc}, categories={all}
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DistributedRedisCluster is the Schema for the distributedredisclusters API
type DistributedRedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DistributedRedisClusterSpec   `json:"spec,omitempty"`
	Status DistributedRedisClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DistributedRedisClusterList contains a list of DistributedRedisCluster
type DistributedRedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DistributedRedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DistributedRedisCluster{}, &DistributedRedisClusterList{})
}
