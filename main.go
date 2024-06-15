package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const SECRET_ENV = "HTWR_UPDATER_WEBHOOK_SECRET"

func EqualHMAC(secret, hash string, payload []byte) bool {
	compare := CreateHMAC(secret, payload)
	return hmac.Equal([]byte(compare), []byte(hash))
}

func CreateHMAC(secret string, payload []byte) string {
	hm := hmac.New(sha256.New, []byte(secret))
	hm.Write(payload)
	return fmt.Sprintf("sha256=%x", hm.Sum(nil))
}

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

	http.HandleFunc("/hooks/update", func(w http.ResponseWriter, r *http.Request) {
		hash := r.Header.Get("X-Hub-Signature-256")

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("No payload or could not read it")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("No payload"))
			return
		}

		// verify that the webhook comes from github
		if !EqualHMAC(os.Getenv(SECRET_ENV), hash, payload) {
			slog.Error("Somebody gave the wrong secret")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		slog.Info("Updating deployment", "name", *namePtr, "namespace", *namespacePtr)

		err = updateDeployment(r.Context(), clientset, *namespacePtr, *namePtr)
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
