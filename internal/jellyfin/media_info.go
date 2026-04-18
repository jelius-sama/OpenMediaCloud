package jellyfin

import (
    "bytes"
    "compress/gzip"
    "encoding/json"
    "io"
    "net/http"
    "strconv"
)

func ApplyMediaInfoPatch(resp *http.Response) error {
    if resp.StatusCode != http.StatusOK {
        return nil
    }

    // Jellyfin may send compressed responses depending on the client's Accept-Encoding header.
    var bodyReader io.Reader = resp.Body
    // NOTE: We don't really need this part anymore
    if resp.Header.Get("Content-Encoding") == "gzip" {
        gzReader, err := gzip.NewReader(resp.Body)
        if err != nil {
            return err
        }
        defer gzReader.Close()
        bodyReader = gzReader
    }

    body, err := io.ReadAll(bodyReader)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Parse the JSON into a generic map so we don't need to model
    // the entire PlaybackInfo structure. We only touch what we need.
    var playbackInfo map[string]any
    if err := json.Unmarshal(body, &playbackInfo); err != nil {
        return err
    }

    // Patch each MediaSource entry.
    mediaSources, ok := playbackInfo["MediaSources"].([]any)
    if !ok {
        // No MediaSources in response, nothing to patch.
        return nil
    }

    for _, source := range mediaSources {
        s, ok := source.(map[string]any)
        if !ok {
            continue
        }

        s["SupportsDirectStream"] = true
        s["SupportsDirectPlay"] = true
        s["SupportsTranscoding"] = false

        s["TranscodingSubProtocol"] = "http"

        // Remove fields that would give the client an HLS path to fall back to.
        delete(s, "TranscodingUrl")
        delete(s, "TranscodingContainer")
    }

    // Re-encode the patched structure back to JSON.
    patched, err := json.Marshal(playbackInfo)
    if err != nil {
        return err
    }

    length := len(patched)
    // Replace the response body with the patched content.
    // We must also update Content-Length to match the new body size,
    // and remove Content-Encoding since we are no longer compressing.
    resp.Body = io.NopCloser(bytes.NewReader(patched))
    resp.ContentLength = int64(length)
    resp.Header.Set("Content-Length", strconv.Itoa(length))
    resp.Header.Del("Content-Encoding")
    resp.Header.Set("Content-Type", "application/json")

    return nil
}

