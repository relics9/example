// Example fix for a Go-based Cloud Run service
func divideHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")

	a, errA := strconv.ParseFloat(aStr, 64)
	if errA != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Parameter 'a' must be a valid number"})
		return
	}

	b, errB := strconv.ParseFloat(bStr, 64)
	if errB != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Parameter 'b' must be a valid number"})
		return
	}

	if b == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Division by zero is not allowed. Parameter 'b' must be non-zero."})
		return
	}

	result := a / b
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]float64{"result": result})
}