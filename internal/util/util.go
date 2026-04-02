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
var videoPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Videos/[^/]+/stream$`),
    regexp.MustCompile(`^/Videos/[^/]+/stream\.[a-zA-Z0-9]+$`),
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
    regexp.MustCompile(`^/videos/[^/]+/master\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/main\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/hls[^/]+/main/-?\d+\.[a-zA-Z0-9]+$`),
}

// /Items/877d0f740648605d91c17d147a9a9ff8/Images/Primary
// /Items/{itemId}/Images/{imageType}
var imagePaths = []*regexp.Regexp{}

// /Items/{itemId}/Download
var downloadPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Items/[^/]+/Download$`),
}

// /Audio/{itemId}/hls/...
// /Audio/{itemId}/stream.{container}
// /Audio/{itemId}/stream
var audioPaths = []*regexp.Regexp{}

type PathKindT = uint8

const (
    PathKindVideos PathKindT = iota
    PathKindMediaInfo
    PathKindDownloads
    PathKindAudios
    PathKindImage
    PathKindHLS
    PathKindDefault
)

func ForwardTo(path string) PathKindT {
    for _, pattern := range videoPaths {
        if pattern.MatchString(path) {
            return PathKindVideos
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

    for _, pattern := range audioPaths {
        if pattern.MatchString(path) {
            return PathKindAudios
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

    if val := os.Getenv("ACCESS_KEY_ID"); len(val) == 0 {
        errCount++
        errs = errs + "\tACCESS_KEY_ID is not set\n"
    }

    if val := os.Getenv("SECRET_ACCESS_KEY"); len(val) == 0 {
        errCount++
        errs = errs + "\tSECRET_ACCESS_KEY is not set\n"
    }

    if val := os.Getenv("ACCOUNT_ID"); len(val) == 0 {
        errCount++
        errs = errs + "\tACCOUNT_ID is not set\n"
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

