package mux

import (
    "net/http"
    "os"

    "github.com/jelius-sama/OpenMediaCloud/internal/jellyfin"
    "github.com/jelius-sama/logger"
)

type Host uint8

// Supported services
const (
    HostJellyfin Host = iota
    HostImmich
    HostKomga
)

func (h Host) ToString() string {
    switch h {
    case HostJellyfin:
        return os.Getenv("UPSTREAM_JELLYFIN_HOST")
    case HostImmich:
        return os.Getenv("UPSTREAM_IMMICH_HOST")
    case HostKomga:
        return os.Getenv("UPSTREAM_KOMGA_HOST")
    }

    logger.Panic("Invalid host enumeration")
    return ""
}

func Multiplexer() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.Host == HostJellyfin.ToString() {
            jellyfin.Router(w, r)
            return
        }

        // TODO: Implement router for immich and komga
        if r.Host == HostImmich.ToString() {
            logger.Warning("It seems that you are using Immich, please note that this project does not yet implement features to support Immich and is in active development. The only usable service currently at this point is Jellyfin")
            http.Error(w, "Feature Not Implemented", http.StatusNotImplemented)
            return
        }

        if r.Host == HostKomga.ToString() {
            logger.Warning("It seems that you are using Immich, please note that this project does not yet implement features to support Immich and is in active development. The only usable service currently at this point is Jellyfin")
            http.Error(w, "Feature Not Implemented", http.StatusNotImplemented)
            return
        }

        logger.Error("Unknown host detected, rejecting client's request")
        http.Error(w, "Something went wrong", http.StatusBadRequest)
    })

    return mux
}

