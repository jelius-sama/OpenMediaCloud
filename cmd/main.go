package main

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strconv"
    "syscall"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/jelius-sama/OpenMediaCloud/internal/db"
    "github.com/jelius-sama/OpenMediaCloud/internal/mux"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"

    "github.com/jelius-sama/logger"
    "github.com/joho/godotenv"
)

const VERSION = "v0.1.2"

var (
    // Set at compile time (use makefile)
    IS_PROD       string
    PORT          string
    CustomEnvPath *string
)

type configWatchDogT struct {
    ActivePath string
}

var configWatchDogC = &configWatchDogT{}

func init() {
    logger.Configure(logger.Cnf{
        IsDev: logger.IsDev{
            EnvironmentVariable: nil,
            ExpectedValue:       nil,
            DirectValue:         logger.BoolPtr(IS_PROD == "FALSE"),
        },
        UseSyslog: false,
    })

    if shouldExit := handleFlags(); shouldExit == true {
        os.Exit(0)
    }

    if CustomEnvPath != nil && len(*CustomEnvPath) != 0 {
        if err := godotenv.Load(*CustomEnvPath); err != nil {
            logger.Fatal(err)
        }
        configWatchDogC.ActivePath = *CustomEnvPath
    } else {
        loadFromEtc := func() error {
            path := filepath.Join("/etc", "OpenMediaCloud", ".env")
            configWatchDogC.ActivePath = path
            return godotenv.Load(path)
        }

        userHome, err := os.UserHomeDir()

        if err != nil {
            logger.Error("Couldn't get user's home directory, loading from `/etc/OpenMediaCloud`.")
            err = loadFromEtc()
            if err != nil {
                logger.Fatal("Error loading environment variables.")
            }
        } else {
            path := filepath.Join(userHome, ".config", "OpenMediaCloud", ".env")
            err = godotenv.Load(path)
            configWatchDogC.ActivePath = path
            if err != nil {
                err = loadFromEtc()
                if err != nil {
                    logger.Fatal("Error loading environment variables.")
                }
            }
        }
    }
}

// TODO: Handle unsetting of environment variables.
// Also consider triggering a "reload" of the watchdog if
// the watcher channel somehow gets killed.
func (w *configWatchDogT) Start() {
    var prevConf []byte
    var err error

    setPrevConf := func() {
        prevConf, err = os.ReadFile(w.ActivePath)
        if err != nil {
            logger.Fatal("failed to read environment file\n\tFile is either deleted or something very serious is wrong.\n\tHow could we manage to read before?")
        }
    }

    setPrevConf() // call at least once

    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        logger.Error("failed to watch environment file, live reloading disabled.")
        return
    }

    defer watcher.Close()
    watcher.Add(w.ActivePath)

    for {
        select {
        case err, ok := <-watcher.Errors:
            if !ok {
                logger.Error("Watcher engine shut down. Live reload is now disabled.")
                return
            }
            logger.Error("config watchdog error:", err)

        case events, ok := <-watcher.Events:
            if !ok {
                logger.Error("Encounter an error, any future changes to environment file will not be applied or watched.")
                return
            }

            if events.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
                watcher.Add(w.ActivePath) // Helps prevent dangling reference

                // Try 10 times, each time with a sleep of 10ms
                for range 10 {
                    if err = godotenv.Overload(w.ActivePath); err != nil {
                        if errors.Is(err, os.ErrNotExist) {
                            time.Sleep(10 * time.Millisecond) // sleep for 10ms
                            continue
                        }
                        logger.Error("Detected a change in", w.ActivePath+".", "\nDue to errors, changes to environment will not be applied.\n\t", err)
                        break
                    }
                    break
                }

                if services, err := util.EnsureENV(); err != nil {
                    envMap, err := godotenv.Parse(bytes.NewReader(prevConf))
                    if err != nil {
                        // NOTE: Broken English:
                        // INFO: The reason why this error should not have happened (unless something catastrophic is wrong with the hardware)
                        // is because, we use the previously set values to set the new value in case of failure. Because we used the previously
                        // set value, the error should normally have happened long before we reached this state (the previous values are incorrect)
                        // Also the error should crash in case the very first config are faulty so there would have been no way we reached till here.
                        logger.Fatal("BUG Encountered, logically this should not have happened but it still did.")
                        // NOTE: For Native English speakers:
                        // INFO: This error should not have occurred under normal circumstances (unless there is a catastrophic
                        // hardware failure). This is because, in the event of a failure, we fall back to the previously set
                        // values to populate the new configuration. Since we are relying on those previously set values, any
                        // error of this nature should have surfaced much earlier in the process — before we ever reached this
                        // state — as it would imply the previous values themselves were already invalid.
                        // Additionally, if the very first configuration was faulty, the program should have crashed at startup,
                        // meaning there would have been no way to reach this point.
                    }

                    for key, value := range envMap {
                        os.Setenv(key, value)
                    }
                    initServices(services)
                } else {
                    setPrevConf()
                    logger.Okay("Detected a change in environment file, successfully updated the configuration")
                }
            }
        }
    }
}

func initServices(services uint8) {
    // jellyfin
    switch (services >> 0) & 1 {
    case 1:
    case 0:
    }

    // immich
    switch (services >> 1) & 1 {
    case 1:
        if db.ImmichConn == nil {
            port, err := strconv.Atoi(os.Getenv("IMMICH_DB_PORT"))
            if err != nil {
                logger.Fatal("Couldn't convert IMMICH_DB_PORT to a valid integer value!")
            }

            err = db.ImmichConnect(db.Config{
                Host:     os.Getenv("IMMICH_DB_HOST"),
                Port:     port,
                Name:     os.Getenv("IMMICH_DB_NAME"),
                User:     os.Getenv("IMMICH_DB_USER"),
                Password: os.Getenv("IMMICH_DB_PASSWORD"),
            })

            if err != nil {
                logger.Fatal(err)
            }
        }

    case 0:
        if db.ImmichConn != nil {
            db.ImmichClose()
        }
    }

    // komga
    switch (services >> 2) & 1 {
    case 1:
    case 2:
    }
}

func main() {
    services, err := util.EnsureENV()
    if err != nil {
        logger.Fatal(err)
    }
    initServices(services)

    go configWatchDogC.Start()

    if keyPair, privKeyPath := os.Getenv("CLOUDFRONT_KEY_PAIR_ID"), os.Getenv("CLOUDFRONT_PRIVATE_KEY_PATH"); len(keyPair) != 0 && len(privKeyPath) != 0 {
        // NOTE: os.Stat doesn't necessarily mean we have read permission.
        file, err := os.OpenFile(privKeyPath, os.O_RDONLY, 0)
        if err != nil {
            logger.Fatal("Failed to read cloudfront private key:", err)
        }
        defer file.Close()
    }

    fmt.Println("\n\033[0;36mOpenMediaCloud", VERSION, "\033[0m")
    logger.Info("Starting server on port", PORT)

    var quit chan os.Signal = make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    var server *http.Server = &http.Server{
        Addr:    PORT,
        Handler: mux.Multiplexer(),
    }

    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server on port "+PORT+"\n", err)
        }
    }()

    <-quit
    var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var deadline, _ = ctx.Deadline()
    var done chan struct{} = make(chan struct{})

    var ticker *time.Ticker = time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    go func() {
        if err := server.Shutdown(ctx); err != nil {
            logger.TimedFatal("Server forced to shutdown:", err)
        }
        close(done)
    }()

    for {
        select {
        case <-done:
            logger.TimedInfo("Server stopped.")
            return

        case <-ctx.Done():
            logger.TimedInfo("Timeout reached:", ctx.Err())
            return

        case <-ticker.C:
            if term := os.Getenv("TERM"); len(term) != 0 {
                // Only show countdown in interactive terminals
                var remaining int = int(time.Until(deadline).Seconds())
                if remaining < 0 {
                    remaining = 0
                }

                fmt.Printf("\r\033[K\033[0;36m[INFO] Shutting down in %d seconds...\033[0m", remaining)
            }
        }
    }
}

