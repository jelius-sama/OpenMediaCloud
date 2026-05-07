package immich

import (
    "github.com/jelius-sama/OpenMediaCloud/internal/util"
    "net/http"
    "os"

    "github.com/jelius-sama/logger"
)

func Router(w http.ResponseWriter, r *http.Request) {
    immichProxy, err := util.MakeReverseProxy(os.Getenv("IMMICH_HOST"))
    if err != nil {
        logger.Panic("[Immich] Failed to make reverse proxy:", err)
    }

    kind, variables := forwardTo(r.URL.Path, r.Method)

    switch kind {
    case pathKindAssetVideo:
        logger.Debug("[TODO] [Immich] Implement Asset Video", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindDownloadArchive:
        logger.Debug("[TODO] [Immich] Implement Download Archive", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindDownloadAsset:
        logger.Debug("[TODO] [Immich] Implement Download Asset", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindViewAsset:
        logger.Debug("[TODO] [Immich] Implement View Asset", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindProfileImage:
        logger.Debug("[TODO] [Immich] Implement Profile Image", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindPersonThumbnail:
        logger.Debug("[TODO] [Immich] Implement Person Thumbnail", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindFallback:
        logger.Info("Forwarding to Immich:", r.Method, r.URL.Path)
        immichProxy.ServeHTTP(w, r)

    default:
        logger.Panic("unreachable")
    }
}

