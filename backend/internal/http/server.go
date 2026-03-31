package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	apiv1alpha1 "smart-cost-optimizer/backend/api/v1alpha1"
	"smart-cost-optimizer/backend/internal/models"
	"smart-cost-optimizer/backend/internal/recommender"
)

type ClusterReader interface {
	ListClusters(ctx context.Context) ([]models.ClusterSummary, error)
}

type ActionService interface {
	List() []models.ActionRecord
	HibernateCluster(context.Context, string) (models.ActionRecord, error)
	ScaleNodePool(context.Context, string, string, int64, int64) (models.ActionRecord, error)
	MoveWorkload(context.Context, string, string, string, string) (models.ActionRecord, error)
}

type Server struct {
	reader         ClusterReader
	engine         *recommender.Engine
	actions        ActionService
	refresh        time.Duration
	frontendOrigin string

	mu       sync.RWMutex
	snapshot models.InventorySnapshot
}

func NewServer(reader ClusterReader, engine *recommender.Engine, actionsService ActionService, refresh time.Duration, frontendOrigin string) *Server {
	return &Server{
		reader:         reader,
		engine:         engine,
		actions:        actionsService,
		refresh:        refresh,
		frontendOrigin: frontendOrigin,
	}
}

func (s *Server) Start(ctx context.Context) {
	s.refreshSnapshot(ctx)

	ticker := time.NewTicker(s.refresh)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.refreshSnapshot(ctx)
			}
		}
	}()
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/v1/clusters", s.handleClusters)
	mux.HandleFunc("/api/v1/recommendations", s.handleRecommendations)
	mux.HandleFunc("/api/v1/recommendations/", s.handleRecommendationByID)
	mux.HandleFunc("/api/v1/actions", s.handleActions)
	mux.HandleFunc("/api/v1/actions/hibernate-cluster", s.handleHibernate)
	mux.HandleFunc("/api/v1/actions/scale-nodepool", s.handleScaleNodePool)
	mux.HandleFunc("/api/v1/actions/move-workload", s.handleMoveWorkload)
	mux.HandleFunc("/api/v1/savings/summary", s.handleSavingsSummary)

	return s.withCORS(mux)
}

func (s *Server) refreshSnapshot(ctx context.Context) {
	clusters, err := s.reader.ListClusters(ctx)
	if err != nil {
		log.Printf("refresh inventory: %v", err)
		return
	}

	snapshot, _, err := s.engine.BuildSnapshot(ctx, clusters)
	if err != nil {
		log.Printf("build snapshot: %v", err)
		return
	}

	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
}

func (s *Server) handleClusters(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	clusters := append([]models.ClusterSummary(nil), s.snapshot.Clusters...)
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, clusters)
}

func (s *Server) handleRecommendations(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	recommendations := append([]models.Recommendation(nil), s.snapshot.Recommendations...)
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, recommendations)
}

func (s *Server) handleRecommendationByID(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(request.URL.Path, "/api/v1/recommendations/")

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, recommendation := range s.snapshot.Recommendations {
		if recommendation.ID == id {
			writeJSON(writer, http.StatusOK, recommendation)
			return
		}
	}

	http.Error(writer, "recommendation not found", http.StatusNotFound)
}

func (s *Server) handleActions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(writer, http.StatusOK, s.actions.List())
}

func (s *Server) handleHibernate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload apiv1alpha1.HibernateClusterRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := s.actions.HibernateCluster(request.Context(), payload.ClusterName)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, record)
		return
	}

	s.refreshSnapshot(request.Context())
	writeJSON(writer, http.StatusAccepted, record)
}

func (s *Server) handleScaleNodePool(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload apiv1alpha1.ScaleNodePoolRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := s.actions.ScaleNodePool(request.Context(), payload.ClusterName, payload.WorkerPool, payload.Minimum, payload.Maximum)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, record)
		return
	}

	s.refreshSnapshot(request.Context())
	writeJSON(writer, http.StatusAccepted, record)
}

func (s *Server) handleMoveWorkload(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload apiv1alpha1.MoveWorkloadRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := s.actions.MoveWorkload(request.Context(), payload.SourceCluster, payload.TargetCluster, payload.Namespace, payload.WorkloadName)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, record)
		return
	}

	s.refreshSnapshot(request.Context())
	writeJSON(writer, http.StatusAccepted, record)
}

func (s *Server) handleSavingsSummary(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	summary := s.snapshot.Summary
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, summary)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", s.frontendOrigin)
		writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func writeJSON(writer http.ResponseWriter, status int, payload interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
