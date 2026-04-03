package util

import (
    "errors"
    "net/http/httputil"
    "net/url"
    "os"
    "regexp"
)

/*
 * EXAMPLE:
 * /Videos/877d0f740648605d91c17d147a9a9ff8/stream
 * /Videos/{id}/stream.{container}
 */
var mediaPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Videos/[^/]+/stream$`),
    regexp.MustCompile(`^/Videos/[^/]+/stream\.[a-zA-Z0-9]+$`),
    regexp.MustCompile(`^/Audio/[^/]+/stream$`),
    regexp.MustCompile(`^/Audio/[^/]+/universal$`),
    regexp.MustCompile(`^/Audio/[^/]+/stream\.[a-zA-Z0-9]+$`),
}

var mediaInfoPath = []*regexp.Regexp{
    regexp.MustCompile(`^/Items/[^/]+/PlaybackInfo$`),
}

/*
 * EXAMPLE:
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/master.m3u8
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/main.m3u8
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/hls1/main/0.mp4
 */
var hlsPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/audio/[^/]+/master\.m3u8$`),
    regexp.MustCompile(`^/audio/[^/]+/main\.m3u8$`),
    regexp.MustCompile(`^/audio/[^/]+/hls[^/]+/main/-?\d+\.[a-zA-Z0-9]+$`),
    regexp.MustCompile(`^/videos/[^/]+/master\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/main\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/hls[^/]+/main/-?\d+\.[a-zA-Z0-9]+$`),
}

// /Items/877d0f740648605d91c17d147a9a9ff8/Images/Primary
// /Items/{itemId}/Images/{imageType}
// INFO: Since image files are relatively small in size I doubt we really
//       need a solution for this part of the problem. This feature may
//       not be implemented or even if it were to be implemented it would be
//       very much at a later point as it is not a priority feature.
/* NOTE: Jellyfin does not really store images in a very accessible place
 * it is stored in it's cache directory, we could just make the cache
 * directory an rclone mount but that may slow things down, this is
 * basically gonna use the same principle as the media files.
 * Another approach is, when a request for an image comes in we serve it
 * and then we check if that specific image exists in R2 bucket with some
 * sort of KeyValue DB and if it doesn't exists we can store the image in
 * our own R2 "cache" folder and then update the KV DB, this was any
 * subsequent request with the same image request signature can be served
 * by redirecting to R2 instead of letting the VPS handle it.
 */
var imagePaths = []*regexp.Regexp{}

// /Items/{itemId}/Download
var downloadPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Items/[^/]+/Download$`),
}

type PathKindT = uint8

const (
    PathKindMedia PathKindT = iota
    PathKindMediaInfo
    PathKindDownloads
    PathKindImage
    PathKindHLS
    PathKindDefault
)

func ForwardTo(path string) PathKindT {
    for _, pattern := range mediaPaths {
        if pattern.MatchString(path) {
            return PathKindMedia
        }
    }

    for _, pattern := range mediaInfoPath {
        if pattern.MatchString(path) {
            return PathKindMediaInfo
        }
    }

    for _, pattern := range downloadPaths {
        if pattern.MatchString(path) {
            return PathKindDownloads
        }
    }

    for _, pattern := range hlsPaths {
        if pattern.MatchString(path) {
            return PathKindHLS
        }
    }

    for _, pattern := range imagePaths {
        if pattern.MatchString(path) {
            return PathKindImage
        }
    }

    return PathKindDefault
}

func MakeReverseProxy(target string) (*httputil.ReverseProxy, error) {
    parsed, err := url.Parse(target)
    if err != nil {
        return nil, err
    }
    return httputil.NewSingleHostReverseProxy(parsed), nil
}

func EnsureENV() error {
    var errs string = "The following environment variables are not set:\n"
    var errCount int = 0

    if val := os.Getenv("JELLYFIN_HOST"); len(val) == 0 {
        errCount++
        errs = errs + "\tJELLYFIN_HOST is not set\n"
    }

    if val := os.Getenv("JELLYFIN_API_KEY"); len(val) == 0 {
        errCount++
        errs = errs + "\tJELLYFIN_API_KEY is not set\n"
    }

    if val := os.Getenv("JELLYFIN_USER_ID"); len(val) == 0 {
        errCount++
        errs = errs + "\tJELLYFIN_USER_ID is not set\n"
    }

    if val := os.Getenv("AWS_REGION"); len(val) == 0 {
        errCount++
        errs = errs + "\tAWS_REGION is not set\n"
    }

    if val := os.Getenv("ACCESS_KEY_ID"); len(val) == 0 {
        errCount++
        errs = errs + "\tACCESS_KEY_ID is not set\n"
    }

    if val := os.Getenv("SECRET_ACCESS_KEY"); len(val) == 0 {
        errCount++
        errs = errs + "\tSECRET_ACCESS_KEY is not set\n"
    }

    if val := os.Getenv("BUCKET_NAME"); len(val) == 0 {
        errCount++
        errs = errs + "\tBUCKET_NAME is not set\n"
    }

    if errCount > 0 {
        return errors.New(errs)
    }

    return nil
}

