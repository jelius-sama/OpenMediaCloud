package util

import (
    "net/http/httputil"
    "net/url"
    "regexp"
)

var videoPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Videos/[^/]+/stream$`),
    regexp.MustCompile(`^/Videos/[^/]+/stream\.[a-zA-Z0-9]+$`),
}

var streamPaths = []*regexp.Regexp{}

var audioPaths = []*regexp.Regexp{}

type PathKindT = int8

const (
    PathKindVideos PathKindT = iota
    PathKindStreams
    PathKindAudios
)

func ShouldForward(path string) (bool, PathKindT) {
    for _, pattern := range videoPaths {
        if pattern.MatchString(path) {
            return true, PathKindVideos
        }
    }

    for _, pattern := range streamPaths {
        if pattern.MatchString(path) {
            return true, PathKindStreams
        }
    }

    for _, pattern := range audioPaths {
        if pattern.MatchString(path) {
            return true, PathKindAudios
        }
    }

    return false, -1
}

func MakeReverseProxy(target string) (*httputil.ReverseProxy, error) {
    parsed, err := url.Parse(target)
    if err != nil {
        return nil, err
    }
    return httputil.NewSingleHostReverseProxy(parsed), nil
}

