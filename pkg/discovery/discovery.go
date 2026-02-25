package discovery

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/discovery"
)

// Features tracks which CRD-based features are available in the cluster.
type Features struct {
	hasOTelOperator     bool
	hasTargetAllocator  bool
	mu                  sync.RWMutex
	ready               bool
	onChange            func(hasOTelOperator, hasTargetAllocator bool)
}

// IsReady returns true after initial discovery is complete.
func (f *Features) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ready
}

// Get returns a snapshot of the current feature state.
func (f *Features) Get() (hasOperator, hasTA bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.hasOTelOperator, f.hasTargetAllocator
}

// CRDWatcher watches for OTel-related CRDs and updates Features.
type CRDWatcher struct {
	discovery discovery.DiscoveryInterface
	features  *Features
	interval  time.Duration
}

// NewCRDWatcher creates a new CRD watcher.
func NewCRDWatcher(disc discovery.DiscoveryInterface, onChange func(hasOTelOperator, hasTargetAllocator bool)) *CRDWatcher {
	return &CRDWatcher{
		discovery: disc,
		features: &Features{
			onChange: onChange,
		},
		interval: 30 * time.Second,
	}
}

// Features returns the current feature state.
func (w *CRDWatcher) Features() *Features {
	return w.features
}

// Start begins the CRD watch loop. It runs until the context is cancelled.
func (w *CRDWatcher) Start(ctx context.Context) {
	// Run initial discovery
	w.discover()
	w.features.mu.Lock()
	w.features.ready = true
	hasOp, hasTA := w.features.hasOTelOperator, w.features.hasTargetAllocator
	w.features.mu.Unlock()
	slog.Info("CRD discovery complete",
		"hasOTelOperator", hasOp,
		"hasTargetAllocator", hasTA,
	)

	// Start periodic re-check
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("CRD watcher stopped")
			return
		case <-ticker.C:
			w.discover()
		}
	}
}

func (w *CRDWatcher) discover() {
	prevOperator, prevTA := w.features.Get()

	hasOperator := false
	hasTA := false

	_, apiResourceLists, err := w.discovery.ServerGroupsAndResources()
	if err != nil {
		slog.Warn("failed to discover server resources, will retry", "error", err)
		return
	}

	for _, list := range apiResourceLists {
		if list == nil {
			continue
		}
		for _, resource := range list.APIResources {
			if resource.Kind == "OpenTelemetryCollector" {
				hasOperator = true
			}
			if resource.Kind == "TargetAllocator" {
				hasTA = true
			}
		}
	}

	w.features.mu.Lock()
	w.features.hasOTelOperator = hasOperator
	w.features.hasTargetAllocator = hasTA
	onChange := w.features.onChange
	w.features.mu.Unlock()

	if hasOperator != prevOperator || hasTA != prevTA {
		slog.Info("CRD feature change detected",
			"hasOTelOperator", hasOperator,
			"hasTargetAllocator", hasTA,
		)
		if onChange != nil {
			onChange(hasOperator, hasTA)
		}
	}
}
