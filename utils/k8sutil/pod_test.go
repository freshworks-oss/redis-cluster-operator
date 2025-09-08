package k8sutil

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
)

func TestNewPodController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewPodController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewPodController() returned nil")
	}

	// Verify it implements IPodControl interface
	_, ok := controller.(IPodControl)
	if !ok {
		t.Error("NewPodController() does not implement IPodControl interface")
	}
}

func TestPodController_CreatePod(t *testing.T) {
	tests := []struct {
		name        string
		pod         *corev1.Pod
		expectError bool
	}{
		{
			name: "Create pod successfully",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
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
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreatePod(tt.pod)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify pod was created
				retrievedPod := &corev1.Pod{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.pod.Namespace,
					Name:      tt.pod.Name,
				}, retrievedPod)
				if getErr != nil {
					t.Errorf("Failed to retrieve pod after creation: %v", getErr)
				}
			}
		})
	}
}

func TestPodController_UpdatePod(t *testing.T) {
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pod",
			Namespace: "test-namespace",
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
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		expectError bool
	}{
		{
			name: "Update existing pod successfully",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-pod",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"app": "updated",
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
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingPod).Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdatePod(tt.pod)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodController_DeletePod(t *testing.T) {
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-to-delete",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		expectError bool
	}{
		{
			name:        "Delete existing pod successfully",
			pod:         existingPod,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingPod).Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePod(tt.pod)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodController_DeletePodByName(t *testing.T) {
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-to-delete",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		podName     string
		expectError bool
	}{
		{
			name:        "Delete existing pod by name successfully",
			namespace:   "test-namespace",
			podName:     "pod-to-delete",
			expectError: false,
		},
		{
			name:        "Delete non-existing pod by name returns error",
			namespace:   "test-namespace",
			podName:     "non-existing-pod",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.podName == "pod-to-delete" {
				fakeClient = fake.NewClientBuilder().WithObjects(existingPod).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePodByName(tt.namespace, tt.podName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodController_GetPod(t *testing.T) {
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		podName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing pod successfully",
			namespace:   "test-namespace",
			podName:     "existing-pod",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing pod returns error",
			namespace:   "test-namespace",
			podName:     "non-existing-pod",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingPod).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			pod, err := controller.GetPod(tt.namespace, tt.podName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && pod == nil {
				t.Error("Expected pod to be found but got nil")
			}
			if tt.expectFound && pod.Name != tt.podName {
				t.Errorf("Expected pod name %s, got %s", tt.podName, pod.Name)
			}
		})
	}
}

func TestPodController_UpdatePodLabels(t *testing.T) {
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"existing": "label",
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
	}

	podWithoutLabels := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-without-labels",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		labelKey    string
		labelValue  string
		expectError bool
	}{
		{
			name:        "Update pod labels successfully",
			pod:         existingPod,
			labelKey:    "app",
			labelValue:  "test-app",
			expectError: false,
		},
		{
			name:        "Add label to pod without labels",
			pod:         podWithoutLabels,
			labelKey:    "app",
			labelValue:  "test-app",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tt.pod).Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdatePodLabels(tt.pod, tt.labelKey, tt.labelValue)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify label was added
			if !tt.expectError && tt.pod.Labels[tt.labelKey] != tt.labelValue {
				t.Errorf("Expected label %s=%s, got %s", tt.labelKey, tt.labelValue, tt.pod.Labels[tt.labelKey])
			}
		})
	}
}

func TestPodController_UpdatePodAnnotations(t *testing.T) {
	podWithAnnotations := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-annotations",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"existing": "annotation",
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
	}

	podWithoutAnnotations := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "pod-without-annotations",
			Namespace:   "test-namespace",
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name            string
		pod             *corev1.Pod
		annotationKey   string
		annotationValue string
		expectError     bool
		expectUpdate    bool
	}{
		{
			name:            "Update existing annotation with different value",
			pod:             podWithAnnotations,
			annotationKey:   "existing",
			annotationValue: "new-value",
			expectError:     false,
			expectUpdate:    true,
		},
		{
			name:            "Don't update annotation with same value",
			pod:             podWithAnnotations,
			annotationKey:   "existing",
			annotationValue: "annotation",
			expectError:     false,
			expectUpdate:    false,
		},
		{
			name:            "Add new annotation",
			pod:             podWithoutAnnotations,
			annotationKey:   "new-key",
			annotationValue: "new-value",
			expectError:     false,
			expectUpdate:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tt.pod).Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdatePodAnnotations(tt.pod, tt.annotationKey, tt.annotationValue)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify annotation was set
			if !tt.expectError && tt.pod.Annotations[tt.annotationKey] != tt.annotationValue {
				t.Errorf("Expected annotation %s=%s, got %s", tt.annotationKey, tt.annotationValue, tt.pod.Annotations[tt.annotationKey])
			}
		})
	}
}

func TestPodController_ListPodsWithLabel(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
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
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	tests := []struct {
		name          string
		namespace     string
		labelKey      string
		labelValue    string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "List pods with matching app label",
			namespace:     "test-namespace",
			labelKey:      "app",
			labelValue:    "test",
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "List pods with specific env label",
			namespace:     "test-namespace",
			labelKey:      "env",
			labelValue:    "dev",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "List pods with no matching labels",
			namespace:     "test-namespace",
			labelKey:      "nonexistent",
			labelValue:    "value",
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(pod1, pod2).Build()
			controller := NewPodController(fakeClient, &testutil.MockMetrics{})

			podList, err := controller.ListPodsWithLabel(tt.namespace, tt.labelKey, tt.labelValue)

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
