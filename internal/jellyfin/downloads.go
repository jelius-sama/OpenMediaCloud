package jellyfin

/*
#include "../../libs/logger/logger.h"
*/
import "C"
import (
    "context"
    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "net/http"
    "net/http/httputil"
    "os"
)

func ApplyDownloadsPatch(w http.ResponseWriter, r *http.Request, originProxy *httputil.ReverseProxy) {
    s3Client := s3.NewS3Client(s3.NewS3ClientT{
        Bucket:          os.Getenv("JELLYFIN_BUCKET_NAME"),
        Region:          os.Getenv("JELLYFIN_AWS_REGION"),
        AccessId:        os.Getenv("JELLYFIN_ACCESS_KEY_ID"),
        SecretAccessKey: os.Getenv("JELLYFIN_SECRET_ACCESS_KEY"),
        BaseURL:         os.Getenv("JELLYFIN_BASE_URL"),
    })
    C.Okay("Caught video download request: " + r.URL.Path)
    r = r.WithContext(context.WithValue(r.Context(), "disposition", "attachment"))

    if err := ApplyPatch(w, r, s3Client); err != nil {
        C.Error(err.Error())
        originProxy.ServeHTTP(w, r)
    }
}

