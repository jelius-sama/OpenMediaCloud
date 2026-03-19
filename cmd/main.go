package main

import (
    "ClientToR2/internal/router"
    "net/http"

    "github.com/jelius-sama/logger"
)

var (
    // Set at compile time (use makefile)
    IS_PROD string
    PORT    string
)

func init() {
    logger.Configure(logger.Cnf{
        IsDev: logger.IsDev{
            EnvironmentVariable: nil,
            ExpectedValue:       nil,
            DirectValue:         logger.BoolPtr(IS_PROD == "FALSE"),
        },
        UseSyslog: false,
    })
}

func main() {
    logger.Info("Starting server on port:", PORT)
    http.ListenAndServe(PORT, router.Router())
}

