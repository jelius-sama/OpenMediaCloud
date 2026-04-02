package handler

import (
    "fmt"
    "net/http"
    "os"
    "strings"

    "github.com/jelius-sama/logger"
)

func extractToken(r *http.Request) string {
    // Check header first
    if token := r.Header.Get("X-Emby-Token"); token != "" {
        return token
    }

    // Check Authorization header (format: "MediaBrowser Token=abc123, ...")
    if auth := r.Header.Get("Authorization"); auth != "" {
        for part := range strings.SplitSeq(auth, ",") {
            part = strings.TrimSpace(part)
            part, _ = strings.CutPrefix(part, "Token=")
            return strings.Trim(part, "\"")
        }
    }

    // Check query parameter
    if token := r.URL.Query().Get("ApiKey"); token != "" {
        return token
    }

    return ""
}

func CheckAuthStatus(r *http.Request) error {
    userId := r.URL.Query().Get("UserId")
    if len(userId) == 0 {
        logger.Warning("Client request to access media does not contain a valid user id.")
    } else {
        logger.Info("User with ID `" + userId + "` is trying to access media.")
    }

    endpoint := fmt.Sprintf("%s/Users/Me", os.Getenv("JELLYFIN_HOST"))

    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return fmt.Errorf("failed to build auth request: %w", err)
    }

    req.Header.Set("X-Emby-Token", extractToken(r))

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to contact Jellyfin server: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Authentication failed [%d].\n\tUser ID: %s", resp.StatusCode, userId)
    }

    return nil
}

