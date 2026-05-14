package immich

/*
#include "../../libs/logger/logger.h"
*/
import "C"
import (
    "io"
    "maps"
    "net/http"
    "os"
    "time"
)

var immichCheckClient = &http.Client{
    Timeout: 10 * time.Second,
}

func authenticate(w http.ResponseWriter, r *http.Request) {
    // Auth: forward the request as-is to Immich and check the status code.
    // Immich enforces all access rules (ownership, album membership, shared
    // links, etc.) so we delegate entirely rather than reimplementing them.
    authReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, os.Getenv("IMMICH_HOST")+r.URL.RequestURI(), nil)
    if err != nil {
        C.Error("[Immich] Failed to build auth check request: " + err.Error())
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    authReq.Header = r.Header.Clone()

    authResp, err := immichCheckClient.Do(authReq)
    if err != nil {
        C.Error("[Immich] Auth check request failed: " + err.Error())
        http.Error(w, "upstream unreachable", http.StatusBadGateway)
        return
    }

    if authResp.StatusCode < 200 || authResp.StatusCode >= 300 {
        // Copy Immich's response verbatim so the client gets the exact
        // error schema it expects (Content-Type, body, status code).
        maps.Copy(w.Header(), authResp.Header)
        w.WriteHeader(authResp.StatusCode)
        io.Copy(w, authResp.Body)
        authResp.Body.Close()
        return
    }

    // Drain the body by reading it into io.Discard
    io.Copy(io.Discard, authResp.Body)
    authResp.Body.Close()
    C.Debug("Auth Passed")
}

