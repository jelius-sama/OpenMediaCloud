package util

import (
    "errors"
    "net/http/httputil"
    "net/url"
    "os"
)

func MakeReverseProxy(target string) (*httputil.ReverseProxy, error) {
    parsed, err := url.Parse(target)
    if err != nil {
        return nil, err
    }
    return httputil.NewSingleHostReverseProxy(parsed), nil
}

// TODO: Cleanup required with the latest introduced UPSTREAM_{feature}_HOST variable.
// HINT: Grouping related variables into one (mutually inclusive/exclusive)
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

    if upstreamJellyfinHost, upstreamImmichHost, upstreamKomgaHost := os.Getenv("UPSTREAM_JELLYFIN_HOST"), os.Getenv("UPSTREAM_IMMICH_HOST"), os.Getenv("UPSTREAM_KOMGA_HOST"); len(upstreamImmichHost) == 0 && len(upstreamJellyfinHost) == 0 && len(upstreamKomgaHost) == 0 {
        errCount++
        errs = errs + "\tNone of the following is set\n" + "\t\tUPSTREAM_JELLYFIN_HOST\n" + "\t\tUPSTREAM_IMMICH_HOST\n" + "\t\tUPSTREAM_KOMGA_HOST\n" + "\tPlease set at least one upstream hostname.\n"
    } else {
        if len(upstreamJellyfinHost) > 0 {
            if err := ValidateHost(upstreamJellyfinHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_JELLYFIN_HOST:\n\t\t" + err.Error() + "\n"
            }
        }

        if len(upstreamImmichHost) > 0 {
            if err := ValidateHost(upstreamImmichHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_KOMGA_HOST:\n\t\t" + err.Error() + "\n"
            }
        }

        if len(upstreamKomgaHost) > 0 {
            if err := ValidateHost(upstreamKomgaHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_KOMGA_HOST:\n\t\t" + err.Error() + "\n"
            }
        }
    }

    // TODO: Add IMMICH_HOST check

    if errCount > 0 {
        return errors.New(errs)
    }

    return nil
}

