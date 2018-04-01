package chainqueryapis

import (
	"net/http"
)

type User struct {

}

func UserNewGet(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
}

