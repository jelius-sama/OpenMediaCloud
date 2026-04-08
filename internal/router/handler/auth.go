package handler

import (
    "fmt"
    "net/http"
    "os"

    "github.com/jelius-sama/logger"
)

func CheckAuthStatus(r *http.Request) error {
    userId := r.URL.Query().Get("UserId")
    if len(userId) == 0 {
        logger.Warning("Client request to access media does not contain a valid user id.")
    } else {
        logger.Info("User with ID `" + userId + "` is trying to access media.")
    }

    req, err := http.NewRequest("GET", fmt.Sprintf("%s/Users/Me", os.Getenv("JELLYFIN_HOST")), nil)
    if err != nil {
        return fmt.Errorf("failed to build auth request: %w", err)
    }

    if token := r.URL.Query().Get("ApiKey"); token != "" {
        logger.Debug("Client token:", token)
        req.Header.Set("X-Emby-Token", token)
    } else {
        for _, headerName := range []string{"Authorization", "X-Emby-Token"} {
            if val := r.Header.Get(headerName); val != "" {
                req.Header.Set(headerName, val)
            }
        }
    }

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

