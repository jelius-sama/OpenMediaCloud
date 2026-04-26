package jellyfin

import (
    "context"
    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "net/http"
    "net/http/httputil"
    "os"

    "github.com/jelius-sama/logger"
)

func ApplyDownloadsPatch(w http.ResponseWriter, r *http.Request, originProxy *httputil.ReverseProxy) {
    s3Client := s3.NewS3Client(os.Getenv("BUCKET_NAME"))
    logger.Okay("Caught video download request:", r.URL.Path)
    r = r.WithContext(context.WithValue(r.Context(), "disposition", "attachment"))

    if err := ApplyPatch(w, r, s3Client); err != nil {
        logger.TimedError(err)
        originProxy.ServeHTTP(w, r)
    }
}

