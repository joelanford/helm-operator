package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// maxLabelValueLength is the maximum allowed length for Kubernetes label values
	maxLabelValueLength = 63
)

// labelSafeKey returns a label-safe version of the key.
// For keys <= 63 characters, it returns the key as-is.
// For longer keys, it returns a 63-character hash.
func labelSafeKey(key string) string {
	if len(key) <= maxLabelValueLength {
		return key
	}

	// Create a hash of the key and use exactly 63 hex characters
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])[:63]
}

func newIndexLabels(owner, key string, rls *release.Release) map[string]string {
	labels := map[string]string{}
	labels["name"] = rls.Name
	labels["owner"] = owner
	labels["status"] = rls.Info.Status.String()
	labels["version"] = strconv.Itoa(rls.Version)
	labels["key"] = labelSafeKey(key)
	labels["type"] = "index"
	return labels
}

func newChunkLabels(owner, key string) map[string]string {
	labels := map[string]string{}
	labels["owner"] = owner
	labels["key"] = labelSafeKey(key)
	labels["type"] = "chunk"
	return labels
}

func newListIndicesLabelSelector(owner string) labels.Selector {
	return labels.Set{"owner": owner, "type": "index"}.AsSelector()
}

// newListAllForKeySelector creates a selector that can find secrets by key,
// handling both original keys and hashed keys for backward compatibility
func newListAllForKeySelector(owner, key string) labels.Selector {
	return labels.Set{"owner": owner, "key": labelSafeKey(key)}.AsSelector()
}

// newListChunksForKeySelector creates a selector that can find chunk secrets by key,
// handling both original keys and hashed keys for backward compatibility
func newListChunksForKeySelector(owner, key string) labels.Selector {
	return labels.Set{"owner": owner, "key": labelSafeKey(key), "type": "chunk"}.AsSelector()
}

var systemLabels = sets.New[string]("name", "owner", "status", "version", "key", "type", "createdAt", "modifiedAt")

// Checks if label is system
func isSystemLabel(key string) bool {
	return systemLabels.Has(key)
}

// Removes system labels from labels map
func filterSystemLabels(lbs map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range lbs {
		if !isSystemLabel(k) {
			result[k] = v
		}
	}
	return result
}
