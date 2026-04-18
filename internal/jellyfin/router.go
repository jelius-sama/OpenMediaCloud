package jellyfin

import (
    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"
    "net/http"
    "os"

    "github.com/jelius-sama/logger"
)

func Router(w http.ResponseWriter, r *http.Request) {
    // NOTE: OR Maybe we can implement all three of the solutions and let the user decide which one to use
    // This however requires us to explicitely state a precedence level of each method as there is a possibility
    // that more than one method are used at a time leading to issues in routing if they are contradicting each other.

    // INFO: Maybe BUT this only applies to those using a master reverse proxy,
    // we can assume that we trust the reverse proxy and the reverse proxy will set a special header
    // which reaching out to our proxy, we can use that token to route the service.
    // This however requires a proxy and cooperation of the proxy.
    // Since I already use Caddy as my master proxy this is not a big deal for me but may not be the case for many.

    // TODO: Identify whether the request is intended for Jellyfin, Immich, or Komga (if I plan to implement Komga).
    // This needs to be done in a host-agnostic way, as users of this proxy can technically use any hostname.
    // This means we may need to either work at the header level, which can potentially be tampered with, OR
    // introduce a config that maps hostnames to specific services, allowing us to pattern match against them.
    // NOTE: hostnames can also be tampered with, so this is not a foolproof solution either.
    jellyfinProxy, err := util.MakeReverseProxy(os.Getenv("JELLYFIN_HOST"))
    if err != nil {
        logger.Panic("Failed to make reverse proxy:", err)
    }

    if (r.Method != http.MethodGet) && (r.Method != http.MethodPost) {
        logger.Info("Forwarding to Jellyfin:", r.Method, r.URL.Path)
        jellyfinProxy.ServeHTTP(w, r)
        return
    }

    kind := util.ForwardTo(r.URL.Path)

    switch kind {
    case util.PathKindMedia:
        if err := CheckAuthStatus(r); err != nil {
            logger.Warning(err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        s3Client := s3.NewS3Client(os.Getenv("BUCKET_NAME"))
        logger.Okay("Caught media request:", r.Method, r.URL.Path)
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
        // TODO: In the future it would be a good idea to handle the errors in our own application so
        //      that we don't have to rely on jellyfin, since error responses are well documented it is
        //      not really that difficult.
        if err := ApplyPatch(w, r, s3Client); err != nil {
            logger.TimedError(err) // INFO: We can temporarily monitor logs and email the admin in case of err.
            // TODO: Implement error handling instead of letting jellyfin do it for us as the error may
            //       be caused due to our implemented of the handler which if it is the case then jellyfin
            //       would end up successfully serving the media to the client costing us egress fees.
            jellyfinProxy.ServeHTTP(w, r)
        }

    case util.PathKindMediaInfo:
        if err := CheckAuthStatus(r); err != nil {
            logger.Warning(err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        logger.Okay("Caught media info request:", r.Method, r.URL.Path)
        originalDirector := jellyfinProxy.Director
        jellyfinProxy.Director = func(req *http.Request) {
            originalDirector(req)
            req.Header.Del("Accept-Encoding")
            req.Header.Set("Accept", "application/json")
        }
        jellyfinProxy.ModifyResponse = ApplyMediaInfoPatch
        jellyfinProxy.ServeHTTP(w, r)

    case util.PathKindHLS:
        if err := CheckAuthStatus(r); err != nil {
            logger.Warning(err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        s3Client := s3.NewS3Client(os.Getenv("BUCKET_NAME"))
        // NOTE: This will break web version of jellyfin, Swiftfin an iOS app for jellyfin works though.
        // FIX: For the above breaking feature, we have implemented media info route interception which
        //        influences the web client to fetch the raw stream instead of HLS everytime, though it has
        //        it's own disadvantages, it works.
        logger.Okay("Caught HLS request:", r.Method, r.URL.Path)
        if err := ApplyPatch(w, r, s3Client); err != nil {
            logger.TimedError(err)
            jellyfinProxy.ServeHTTP(w, r)
        }

    case util.PathKindDownloads:
        if err := CheckAuthStatus(r); err != nil {
            logger.Warning(err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        logger.Okay("Caught download request:", r.Method, r.URL.Path)
        ApplyDownloadsPatch(w, r, jellyfinProxy)

    case util.PathKindImage:
        logger.Debug("Don't forget to check for auth status.")
        logger.Okay("TODO: Caught image request:", r.Method, r.URL.Path)
        // ApplyImagePatch(r)
        jellyfinProxy.ServeHTTP(w, r)

    case util.PathKindDefault:
        logger.Info("Forwarding to Jellyfin:", r.Method, r.URL.Path)
        jellyfinProxy.ServeHTTP(w, r)

    default:
        logger.Panic("unreachable")
    }
}

