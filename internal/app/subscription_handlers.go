package app

import "net/http"

func registerSubscriptionNodeRoutes(router *http.ServeMux, service SubscriptionNodeService) {
	if service == nil {
		return
	}
	router.HandleFunc("GET /api/v1/subscriptions/nodes", func(writer http.ResponseWriter, request *http.Request) {
		sources, err := service.List(request.Context())
		if err != nil {
			writeAPIError(writer, http.StatusInternalServerError, "subscription_cache_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"sources": sources})
	})
}
