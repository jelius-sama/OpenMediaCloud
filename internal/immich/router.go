package immich

import (
    "net/http"
    "os"

    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"

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
        // Browse page:
        // src="/api/assets/23a8e7e6-993a-4001-bbe9-a0814b73bf77/thumbnail?size=thumbnail&c=JfgNFIKJh3Z%2Fh3d5iHV0gGoExw%3D%3D&edited=true"

        // Dedicated page
        // https://photos.jelius.dev/api/assets/3a2c36b5-ba41-47c1-a693-b290b76c6130/video/playback?c=YAgGFIIQbT6tZFxZdlZ7YAcVmQ==

        // [DEBUG] matchPattern()
        //         str: /api/assets/ff147990-99b4-4dd3-a0b4-1a7c9f874159/video/playback
        //         pattern: /api/people/{id}/thumbnail
        //
        // [DEBUG] matchPattern()
        //         str: /api/assets/ff147990-99b4-4dd3-a0b4-1a7c9f874159/video/playback
        //         pattern: /api/users/{id}/profile-image
        //
        // [DEBUG] matchPattern()
        //         str: /api/assets/ff147990-99b4-4dd3-a0b4-1a7c9f874159/video/playback
        //         pattern: /api/assets/{id}/original
        //
        // [DEBUG] matchPattern()
        //         str: /api/assets/ff147990-99b4-4dd3-a0b4-1a7c9f874159/video/playback
        //         pattern: /api/assets/{id}/thumbnail
        //
        // [DEBUG] matchPattern()
        //         str: /api/assets/ff147990-99b4-4dd3-a0b4-1a7c9f874159/video/playback
        //         pattern: /api/assets/{id}/video/playback
        //
        // [DEBUG] [TODO] [Immich] Implement Asset Video [ff147990-99b4-4dd3-a0b4-1a7c9f874159]
        logger.Debug("[TODO] [Immich] Implement Asset Video", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindDownloadArchive:
        logger.Debug("[TODO] [Immich] Implement Download Archive", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindDownloadAsset:
        logger.Debug("[TODO] [Immich] Implement Download Asset", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindViewAsset:
        s3Client := s3.NewS3Client(s3.NewS3ClientT{
            Bucket:          os.Getenv("IMMICH_BUCKET_NAME"),
            Region:          os.Getenv("IMMICH_AWS_REGION"),
            AccessId:        os.Getenv("IMMICH_ACCESS_KEY_ID"),
            SecretAccessKey: os.Getenv("IMMICH_SECRET_ACCESS_KEY"),
            BaseURL:         os.Getenv("IMMICH_BASE_URL"),
        })
        if err := viewAsset(w, r, variables, s3Client); err != nil {
            logger.Error("[Immich]", err)
            immichProxy.ServeHTTP(w, r)
        }

    case pathKindProfileImage:
        // [DEBUG] matchPattern()
        //         str: /api/users/c9e33b60-d0f5-43e4-b890-5a4fba4b2558/profile-image
        //         pattern: /api/people/{id}/thumbnail
        //
        // [DEBUG] matchPattern()
        //         str: /api/users/c9e33b60-d0f5-43e4-b890-5a4fba4b2558/profile-image
        //         pattern: /api/users/{id}/profile-image
        //
        // [DEBUG] [TODO] [Immich] Implement Profile Image [c9e33b60-d0f5-43e4-b890-5a4fba4b2558]
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

