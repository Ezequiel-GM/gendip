package gendip

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/gorilla/mux"
	"github.com/zond/godip/common"
	"github.com/zond/godip/variants"
	"google.golang.org/appengine"
)

func init() {
	functions.HTTP("resolve", resolve);
}

func corsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}

func preflight(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
}

func resolve(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	requestBody := &common.RequestBody{}
	if err := json.NewDecoder(r.Body).Decode(requestBody); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	p := requestBody.Phase
	variant := variants.FromJson(requestBody.VariantData)
	state, err := p.State(variant)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err = state.Next(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Load the new godip phase from the state
	nextPhase := common.NewPhase(state)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err = json.NewEncoder(w).Encode(nextPhase); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func start(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	variantName := mux.Vars(r)["variant"]
	variant, found := variants.Variants[variantName]
	if !found {
		http.Error(w, fmt.Sprintf("Variant %q not found", variantName), 404)
		return
	}
	state, err := variant.Start()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	phase := common.NewPhase(state)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err = json.NewEncoder(w).Encode(phase); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func listVariants(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(variants.Variants); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func main() {
	r := mux.NewRouter()
	r.Methods("OPTIONS").HandlerFunc(preflight)
	variants := r.Path("/{variant}").Subrouter()
	variants.Methods("POST").HandlerFunc(resolve)
	variants.Methods("GET").HandlerFunc(start)
	r.Path("/").HandlerFunc(listVariants)
	http.Handle("/", r)
	appengine.Main()
}
