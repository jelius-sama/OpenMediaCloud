package mux

/*
#include "../../libs/logger/logger.h"
*/
import "C"
import (
    "net/http"
    "os"

    "github.com/jelius-sama/OpenMediaCloud/internal/immich"
    "github.com/jelius-sama/OpenMediaCloud/internal/jellyfin"
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

    C.Panic("Invalid host enumeration")
    return ""
}

func Multiplexer() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.Host == HostJellyfin.ToString() {
            jellyfin.Router(w, r)
            return
        }

        if r.Host == HostImmich.ToString() {
            immich.Router(w, r)
            return
        }

        // TODO: Implement router for komga
        if r.Host == HostKomga.ToString() {
            C.Warn("It seems that you are using Komga, please note that this project does not yet implement features to support Komga and is in active development. The only usable service currently at this point is Jellyfin")
            http.Error(w, "Feature Not Implemented", http.StatusNotImplemented)
            return
        }

        C.Error("Unknown host detected, rejecting client's request")
        http.Error(w, "Something went wrong", http.StatusBadRequest)
    })

    return mux
}

