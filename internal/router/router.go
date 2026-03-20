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

        mediaProxy, err := util.MakeReverseProxy("http://localhost:6969")
        if err != nil {
            logger.Panic("Failed to make reverse proxy:", err)
        }

        jellyfinProxy, err := util.MakeReverseProxy(jellyfinHost)
        if err != nil {
            logger.Panic("Failed to make reverse proxy:", err)
        }

        found, where := util.ShouldForward(r.URL.Path)
        if found {
            switch where {
            case util.PathKindVideos:
                logger.Okay("Caught video request:", r.URL.Path)
                handler.ApplyVideosPatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindStreams:
                logger.Okay("Caught stream request:", r.URL.Path)
                handler.ApplyStreamsPatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindAudios:
                logger.Okay("Caught audio request:", r.URL.Path)
                handler.ApplyAudiosPatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindImage:
                logger.Okay("Caught image request:", r.URL.Path)
                handler.ApplyImagePatch(r)
                mediaProxy.ServeHTTP(w, r)

            case util.PathKindHLS:
                logger.Okay("Caught HLS request:", r.URL.Path)
                handler.ApplyHLSPatch(r)
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

