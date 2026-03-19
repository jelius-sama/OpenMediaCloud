package router

import (
    "ClientToR2/internal/router/handler"
    "ClientToR2/internal/util"
    "net/http"
    "os"

    "github.com/jelius-sama/logger"
)

func Router() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        jellyfinHost := os.Getenv("JELLYFIN_HOST")
        if jellyfinHost == "" {
            logger.Fatal("JELLYFIN_HOST environment variable is not set")
        }

        // Some features are not implemented so just redirect it back to jellyfin
        // don't worry there is no infinite loop here because it doesn't go though
        // the reverse proxy and directly reaches out to the jellyfin server.
        mediaProxy, err := util.MakeReverseProxy("http://localhost:8096")
        if err != nil {
            logger.Panic("Failed to make reverse proxy:", err)
        }

        jellyfinProxy, err := util.MakeReverseProxy(jellyfinHost)
        if err != nil {
            logger.Panic("Failed to make reverse proxy:", err)
        }

        found, where := util.ShouldForward(r.URL.Path)
        if found {
            logger.Okay("Caught media request:", r.URL.Path)

            switch where {
            case util.PathKindVideos:
                handler.ApplyVideosPatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindStreams:
                handler.ApplyStreamsPatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindAudios:
                handler.ApplyAudiosPatch(r)
                mediaProxy.ServeHTTP(w, r)

            default:
                logger.Panic("unreachable")
            }
        } else {
            logger.Info("Forwarding to Jellyfin:", r.URL.Path)
            jellyfinProxy.ServeHTTP(w, r)
        }
    })

    return mux
}

