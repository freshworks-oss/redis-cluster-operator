package k8sutil

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
)

func TestNewStatefulSetController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewStatefulSetController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewStatefulSetController() returned nil")
	}

	// Verify it implements IStatefulSetControl interface
	_, ok := controller.(IStatefulSetControl)
	if !ok {
		t.Error("NewStatefulSetController() does not implement IStatefulSetControl interface")
	}
}

func TestStatefulSetController_CreateStatefulSet(t *testing.T) {
	tests := []struct {
		name        string
		statefulSet *appsv1.StatefulSet
		expectError bool
	}{
		{
			name: "Create StatefulSet successfully",
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-statefulset",
					Namespace: "test-namespace",
				},
				Spec: appsv1.StatefulSetSpec{
					ServiceName: "test-service",
					Replicas:    int32Ptr(3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateStatefulSet(tt.statefulSet)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify StatefulSet was created
				retrievedSTS := &appsv1.StatefulSet{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.statefulSet.Namespace,
					Name:      tt.statefulSet.Name,
				}, retrievedSTS)
				if getErr != nil {
					t.Errorf("Failed to retrieve StatefulSet after creation: %v", getErr)
				}
			}
		})
	}
}

func TestStatefulSetController_UpdateStatefulSet(t *testing.T) {
	existingSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-statefulset",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		statefulSet *appsv1.StatefulSet
		expectError bool
	}{
		{
			name: "Update existing StatefulSet successfully",
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-statefulset",
					Namespace: "test-namespace",
				},
				Spec: appsv1.StatefulSetSpec{
					ServiceName: "test-service",
					Replicas:    int32Ptr(5),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "updated-container",
									Image: "updated-image",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingSTS).Build()
			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateStatefulSet(tt.statefulSet)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestStatefulSetController_DeleteStatefulSet(t *testing.T) {
	existingSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset-to-delete",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		statefulSet *appsv1.StatefulSet
		expectError bool
	}{
		{
			name:        "Delete existing StatefulSet successfully",
			statefulSet: existingSTS,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingSTS).Build()
			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteStatefulSet(tt.statefulSet)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestStatefulSetController_DeleteStatefulSetByName(t *testing.T) {
	existingSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset-to-delete",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		stsName     string
		expectError bool
	}{
		{
			name:        "Delete existing StatefulSet by name successfully",
			namespace:   "test-namespace",
			stsName:     "statefulset-to-delete",
			expectError: false,
		},
		{
			name:        "Delete non-existing StatefulSet by name returns error",
			namespace:   "test-namespace",
			stsName:     "non-existing-statefulset",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.stsName == "statefulset-to-delete" {
				fakeClient = fake.NewClientBuilder().WithObjects(existingSTS).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteStatefulSetByName(tt.namespace, tt.stsName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestStatefulSetController_GetStatefulSet(t *testing.T) {
	existingSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-statefulset",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		stsName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing StatefulSet successfully",
			namespace:   "test-namespace",
			stsName:     "existing-statefulset",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing StatefulSet returns error",
			namespace:   "test-namespace",
			stsName:     "non-existing-statefulset",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingSTS).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			sts, err := controller.GetStatefulSet(tt.namespace, tt.stsName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && sts == nil {
				t.Error("Expected StatefulSet to be found but got nil")
			}
			if tt.expectFound && sts.Name != tt.stsName {
				t.Errorf("Expected StatefulSet name %s, got %s", tt.stsName, sts.Name)
			}
		})
	}
}

func TestStatefulSetController_ListStatefulSetByLabels(t *testing.T) {
	sts1 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sts1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	sts2 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sts2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(5),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		namespace     string
		labels        map[string]string
		expectedCount int
		expectError   bool
	}{
		{
			name:      "List StatefulSets with matching app label",
			namespace: "test-namespace",
			labels: map[string]string{
				"app": "test",
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:      "List StatefulSets with specific env label",
			namespace: "test-namespace",
			labels: map[string]string{
				"env": "dev",
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:      "List StatefulSets with no matching labels",
			namespace: "test-namespace",
			labels: map[string]string{
				"nonexistent": "value",
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(sts1, sts2).Build()
			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			stsList, err := controller.ListStatefulSetByLabels(tt.namespace, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(stsList.Items) != tt.expectedCount {
				t.Errorf("Expected %d StatefulSets, got %d", tt.expectedCount, len(stsList.Items))
			}
		})
	}
}

func TestStatefulSetController_GetStatefulSetPods(t *testing.T) {
	existingSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "test-service",
			Replicas:    int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts-0",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
	}

	tests := []struct {
		name          string
		namespace     string
		stsName       string
		expectError   bool
		expectedCount int
	}{
		{
			name:          "Get pods for existing StatefulSet",
			namespace:     "test-namespace",
			stsName:       "test-sts",
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:        "Get pods for non-existing StatefulSet returns error",
			namespace:   "test-namespace",
			stsName:     "non-existing-sts",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.stsName == "test-sts" {
				fakeClient = fake.NewClientBuilder().WithObjects(existingSTS, pod1, pod2).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			podList, err := controller.GetStatefulSetPods(tt.namespace, tt.stsName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(podList.Items) != tt.expectedCount {
				t.Errorf("Expected %d pods, got %d", tt.expectedCount, len(podList.Items))
			}
		})
	}
}

func TestStatefulSetController_GetStatefulSetPodsByLabels(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
	}

	tests := []struct {
		name          string
		namespace     string
		labels        map[string]string
		expectedCount int
		expectError   bool
	}{
		{
			name:      "Get pods with matching app label",
			namespace: "test-namespace",
			labels: map[string]string{
				"app": "test",
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:      "Get pods with specific env label",
			namespace: "test-namespace",
			labels: map[string]string{
				"env": "dev",
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:      "Get pods with no matching labels",
			namespace: "test-namespace",
			labels: map[string]string{
				"nonexistent": "value",
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(pod1, pod2).Build()
			controller := NewStatefulSetController(fakeClient, &testutil.MockMetrics{})

			podList, err := controller.GetStatefulSetPodsByLabels(tt.namespace, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(podList.Items) != tt.expectedCount {
				t.Errorf("Expected %d pods, got %d", tt.expectedCount, len(podList.Items))
			}
		})
	}
}

// Helper function to create int32 pointer
