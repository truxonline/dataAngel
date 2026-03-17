package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionv1.AddToScheme(runtimeScheme)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if _, _, err := deserializer.Decode(body, nil, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding admission review: %v", err), http.StatusBadRequest)
		return
	}

	request := admissionReview.Request
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshaling pod: %v", err), http.StatusBadRequest)
		return
	}

	// Check for DataGuard annotations
	patchOperations := []map[string]interface{}{}

	// Inject rclone sidecar if annotation is present
	if _, ok := pod.Annotations["dataguard/rclone-sidecar"]; ok {
		patchOperations = append(patchOperations, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/containers/-",
			"value": createRcloneSidecar(pod),
		})
	}

	// Inject litestream sidecar if annotation is present
	if _, ok := pod.Annotations["dataguard/litestream-sidecar"]; ok {
		patchOperations = append(patchOperations, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/containers/-",
			"value": createLitestreamSidecar(pod),
		})
	}

	// Create admission response
	patchBytes, _ := json.Marshal(patchOperations)
	patchType := admissionv1.PatchTypeJSONPatch

	admissionReview.Response = &admissionv1.AdmissionResponse{
		UID:       request.UID,
		Allowed:   true,
		Patch:     &patchBytes,
		PatchType: &patchType,
	}

	responseBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

func createRcloneSidecar(pod corev1.Pod) corev1.Container {
	bucket := pod.Annotations["dataguard/rclone-bucket"]
	if bucket == "" {
		bucket = "default-bucket"
	}

	return corev1.Container{
		Name:            "rclone-sidecar",
		Image:           "charchess/sidecar-rclone:latest",
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
				ReadOnly:  true,
			},
		},
		Args: []string{
			"sync",
			"/data",
			fmt.Sprintf("s3:%s", bucket),
			"--transfers", "4",
			"--checkers", "8",
			"--stats", "60s",
		},
	}
}

func createLitestreamSidecar(pod corev1.Pod) corev1.Container {
	bucket := pod.Annotations["dataguard/litestream-bucket"]
	if bucket == "" {
		bucket = "default-bucket"
	}

	dbPath := pod.Annotations["dataguard/litestream-db-path"]
	if dbPath == "" {
		dbPath = "/data/app.db"
	}

	return corev1.Container{
		Name:            "litestream-sidecar",
		Image:           "charchess/sidecar-litestream:latest",
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
				ReadOnly:  false,
			},
		},
		Args: []string{
			"replicate",
			dbPath,
			fmt.Sprintf("s3://%s/backups", bucket),
		},
	}
}

func main() {
	http.HandleFunc("/healthz", handleHealthz)
	http.HandleFunc("/mutate", handleMutate)

	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8443"
	}

	log.Printf("Starting webhook server on port %s", port)
	log.Fatal(http.ListenAndServeTLS(":"+port, "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key", nil))
}
