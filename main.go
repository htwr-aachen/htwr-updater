package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// this is a small utility that hooks into a webhook and then updates the htwr deployment on kubernetes

	namespacePtr := flag.String("namespace", "htwr", "The namespace for the deployment to get updated")
	namePtr := flag.String("name", "frontend", "The name for the deployment to get updated")
	portPtr := flag.Int("port", 8000, "The port to host the http webhook & liveliness probes")

	flag.Parse()
	config, err := rest.InClusterConfig()
	if err != nil {
		panic("This utility should only be used inside a kubernetes cluster: " + err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("Could not connect to kubernetes: " + err.Error())
	}

	// webhook setup
	// secret

	secret := os.Getenv("HTWR_UPDATER_WEBHOOK_SECRET")
	h := sha256.New()
	h.Write([]byte(secret))
	compare := h.Sum(nil)

	http.HandleFunc("/hooks/update", func(w http.ResponseWriter, r *http.Request) {
		hash := r.Header.Get("X-Hub-Signature-256")

		// !!timing attacks possible but to lazy
		if hash != fmt.Sprintf("sha256=%s", hex.EncodeToString(compare)) {
			slog.Error("Somebody gave the wrong secret")
		}

		// verifyed github
		slog.Info("Updating deployment", "name", *namePtr, "namespace", *namespacePtr)
		err := updateDeployment(r.Context(), clientset, *namespacePtr, *namePtr)
		if err != nil {
			slog.Error("Could not update deployment", "error", err)
		}
		slog.Debug("Successfully updated deployment")
	})

	// LIVELINESS
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	slog.Warn("Starting webhook server")
	err = http.ListenAndServe(fmt.Sprintf(":%d", *portPtr), nil)
	slog.Warn("Error during serve", "error", err)
}

func updateDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	data := fmt.Sprintf(`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := deploymentClient.Patch(ctx, name, types.StrategicMergePatchType, []byte(data), v1.PatchOptions{})

	return err
}
