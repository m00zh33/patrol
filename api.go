package patrol

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/streadway/handy/accept"
	"github.com/streadway/handy/encoding"
)

// API implements the Patrol service HTTP API.
type API struct {
	log  *log.Logger
	repo Repo
}

// NewAPI returns a new Patrol API.
func NewAPI(l *log.Logger, repo Repo) *API {
	return &API{
		log:  l,
		repo: repo,
	}
}

// Handler returns the http.Handler of the API.
func (api *API) Handler() http.Handler {
	mediaTypes := []string{"application/x-gob", "application/json"}
	handleBuckets := accept.Middleware(mediaTypes...)(
		encoding.GzipTypes(mediaTypes, http.HandlerFunc(api.handleBuckets)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/take" && r.Method == http.MethodPost:
			api.handleTake(w, r)
		case r.URL.Path == "/buckets" && r.Method == http.MethodGet:
			handleBuckets.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// handler for GET /buckets
func (api *API) handleBuckets(w http.ResponseWriter, r *http.Request) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Accept"))
	if err != nil {
		mt = "application/json"
	}

	buckets, err := api.repo.GetBuckets(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		api.log.Printf("handleBuckets: repo error: %v", err)
		return
	}

	var buf bytes.Buffer
	switch mt {
	default:
		fallthrough
	case "application/json":
		err = json.NewEncoder(&buf).Encode(buckets)
	case "application/x-gob":
		err = gob.NewEncoder(&buf).Encode(buckets)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		api.log.Printf("handleBuckets: encoding error: %v", err)
		return
	}

	w.Header().Set("Content-Type", mt)
	buf.WriteTo(w)
}

// handler for POST /take?bucket=my-bucket-name&count=1&rate=100:1s
func (api *API) handleTake(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	name := q.Get("bucket")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		api.log.Print("handleTake: empty bucket name")
		return
	}

	rate, err := ParseRate(q.Get("rate"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		api.log.Printf("handleTake: parse rate error: %v", err)
		return
	}

	count, err := strconv.ParseUint(q.Get("count"), 10, 64)
	if err != nil {
		count = 1
	}

	bucket, err := api.repo.GetBucket(r.Context(), name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		api.log.Printf("handleTake: repo.GetBucket error: %v", err)
		return
	}

	ok := bucket.Take(time.Now(), rate, count)

	if err = api.repo.UpdateBucket(r.Context(), name, bucket); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		api.log.Printf("handleTake: repo.UpdateBucket error: %v", err)
		return
	}

	if !ok {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
}