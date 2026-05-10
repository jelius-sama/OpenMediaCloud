package immich

import (
    "context"
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
        authenticate(w, r)
        s3Client := s3.NewS3Client(s3.NewS3ClientT{
            Bucket:          os.Getenv("IMMICH_BUCKET_NAME"),
            Region:          os.Getenv("IMMICH_AWS_REGION"),
            AccessId:        os.Getenv("IMMICH_ACCESS_KEY_ID"),
            SecretAccessKey: os.Getenv("IMMICH_SECRET_ACCESS_KEY"),
            BaseURL:         os.Getenv("IMMICH_BASE_URL"),
        })
        r = r.WithContext(context.WithValue(r.Context(), "type", ATVideo.String()))
        if err := viewAsset(w, r, variables, s3Client); err != nil {
            logger.Error("[Immich]", err)
            immichProxy.ServeHTTP(w, r)
        }

    case pathKindDownloadAsset:
        authenticate(w, r)
        s3Client := s3.NewS3Client(s3.NewS3ClientT{
            Bucket:          os.Getenv("IMMICH_BUCKET_NAME"),
            Region:          os.Getenv("IMMICH_AWS_REGION"),
            AccessId:        os.Getenv("IMMICH_ACCESS_KEY_ID"),
            SecretAccessKey: os.Getenv("IMMICH_SECRET_ACCESS_KEY"),
            BaseURL:         os.Getenv("IMMICH_BASE_URL"),
        })
        r = r.WithContext(context.WithValue(r.Context(), "disposition", "attachment"))
        if err := viewAsset(w, r, variables, s3Client); err != nil {
            logger.Error("[Immich]", err)
            immichProxy.ServeHTTP(w, r)
        }

    case pathKindViewAsset:
        authenticate(w, r)
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
        authenticate(w, r)
        s3Client := s3.NewS3Client(s3.NewS3ClientT{
            Bucket:          os.Getenv("IMMICH_BUCKET_NAME"),
            Region:          os.Getenv("IMMICH_AWS_REGION"),
            AccessId:        os.Getenv("IMMICH_ACCESS_KEY_ID"),
            SecretAccessKey: os.Getenv("IMMICH_SECRET_ACCESS_KEY"),
            BaseURL:         os.Getenv("IMMICH_BASE_URL"),
        })
        if err := getProfileImage(w, r, variables, s3Client); err != nil {
            logger.Error("[Immich]", err)
            immichProxy.ServeHTTP(w, r)
        }

    case pathKindPersonThumbnail:
        authenticate(w, r)
        s3Client := s3.NewS3Client(s3.NewS3ClientT{
            Bucket:          os.Getenv("IMMICH_BUCKET_NAME"),
            Region:          os.Getenv("IMMICH_AWS_REGION"),
            AccessId:        os.Getenv("IMMICH_ACCESS_KEY_ID"),
            SecretAccessKey: os.Getenv("IMMICH_SECRET_ACCESS_KEY"),
            BaseURL:         os.Getenv("IMMICH_BASE_URL"),
        })
        if err := getPersonThumbnail(w, r, variables, s3Client); err != nil {
            logger.Error("[Immich]", err)
            immichProxy.ServeHTTP(w, r)
        }

    case pathKindDownloadArchive:
        // authenticate(w, r)
        logger.Debug("[TODO] [Immich] Implement Download Archive", variables)
        immichProxy.ServeHTTP(w, r)

    case pathKindFallback:
        logger.Info("Forwarding to Immich:", r.Method, r.URL.Path)
        immichProxy.ServeHTTP(w, r)

    default:
        logger.Panic("unreachable")
    }
}

