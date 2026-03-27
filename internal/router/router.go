package router

import (
    "ClientToR2/internal/router/handler"
    "ClientToR2/internal/s3"
    "ClientToR2/internal/util"
    "net/http"
    "os"

    "github.com/jelius-sama/logger"
)

func Router() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        jellyfinHost := os.Getenv("JELLYFIN_HOST")

        jellyfinProxy, err := util.MakeReverseProxy(jellyfinHost)
        if err != nil {
            logger.Panic("Failed to make reverse proxy:", err)
        }

        found, where := util.ShouldForward(r.URL.Path)
        s3Client := s3.NewS3Client(os.Getenv("BUCKET_NAME"))

        if found {
            switch where {
            case util.PathKindVideos:
                logger.Okay("Caught video request:", r.URL.Path)
                // NOTE: If the handler encountered an error it means two things:
                //  1. Either jellyfin server has updated their API and our proxy failed to communicate.
                //  2. Or our handler function has an edge case that we are not handling well.
                //  3. It could also be network issues when calling to R2 API.
                // INFO: Either way, in case of an error we can basically do one of two things:
                //       It could really be an error that even jellyfin might not be able to handle
                //       or it is an edge case in our implementation, what we can do is forward the
                //       request to jellyfin server for it to handle it.
                //       Now there is one issue with this method, before we mention the issue, below is an
                //       alternative method: We can basically mimic the response that jellyfin would've sent
                //       in case of an error and it is well documented in their API documentations.
                //       Now, lets look at the issue, the problem with forwarding failed handler to jellyfin
                //       is that we don't know if the failure happened because of our implementation or not
                //       therefore if it was a problem with our implementation then jellyfin server would
                //       successfully be able to handle the client request thereby costing us EC2 EGRESS usage.
                //       If there really was an error though, forwarding to jellyfin would handle the error
                //       for us and we do not have to care BUT it might just be costing us egress fee.
                // FIX: In the future it would be a good idea to handle the errors in our own application so
                //      that we don't have to rely on jellyfin, since error responses are well documented it is
                //      not really that difficult.
                if err := handler.ApplyVideosPatch(w, r, s3Client); err != nil {
                    logger.TimedError(err) // INFO: We can temporarily monitor logs and email the admin in case of err.
                    // TODO: Implement error handling instead of letting jellyfin do it for us as the error may
                    //       be caused due to our implemented of the handler which if it is the case then jellyfin
                    //       would end up successfully serving the media to the client costing us egress fees.
                    jellyfinProxy.ServeHTTP(w, r)
                }

            case util.PathKindHLS:
                // NOTE: This will break web version of jellyfin, Swiftfin an iOS app for jellyfin works though.
                logger.Okay("Caught HLS request:", r.URL.Path)
                if err := handler.ApplyVideosPatch(w, r, s3Client); err != nil {
                    logger.TimedError(err)
                    jellyfinProxy.ServeHTTP(w, r)
                }

            case util.PathKindDownloads:
                logger.Okay("Caught stream request:", r.URL.Path)
                handler.ApplyDownloadsPatch(r)
                jellyfinProxy.ServeHTTP(w, r)

            case util.PathKindAudios:
                logger.Okay("Caught audio request:", r.URL.Path)
                handler.ApplyAudiosPatch(r)
                jellyfinProxy.ServeHTTP(w, r)

            case util.PathKindImage:
                logger.Okay("Caught image request:", r.URL.Path)
                handler.ApplyImagePatch(r)
                jellyfinProxy.ServeHTTP(w, r)

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

