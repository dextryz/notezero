package notezero

import (
	"fmt"
	"net/http"

	"github.com/nbd-wtf/go-nostr/nip19"
)

func (s *Handler) RedirectSearch(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("search")
	fmt.Printf("Search: %s\n", code)
	http.Redirect(w, r, "/"+code, http.StatusFound)
}

func (s *Handler) RedirectFromPSlash(w http.ResponseWriter, r *http.Request) {
	code, _ := nip19.EncodePublicKey(r.URL.Path[3:])
	http.Redirect(w, r, "/"+code, http.StatusFound)
}
