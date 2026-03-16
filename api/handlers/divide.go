func (h *Handler) Divide(w http.ResponseWriter, r *http.Request) {
    a := r.URL.Query().Get("a")
    b := r.URL.Query().Get("b")
    
    aVal, err := strconv.ParseFloat(a, 64)
    if err != nil {
        http.Error(w, "invalid parameter 'a'", http.StatusBadRequest)
        return
    }
    
    bVal, err := strconv.ParseFloat(b, 64)
    if err != nil {
        http.Error(w, "invalid parameter 'b'", http.StatusBadRequest)
        return
    }
    
    // Add validation to prevent division by zero
    if bVal == 0 {
        http.Error(w, "divisor cannot be zero", http.StatusBadRequest)
        return
    }
    
    result := aVal / bVal
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]float64{"result": result})
}