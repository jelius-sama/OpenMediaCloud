package main

import (
    "bytes"
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/jelius-sama/OpenMediaCloud/internal/mux"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"

    "github.com/jelius-sama/logger"
    "github.com/joho/godotenv"
)

const VERSION = "v3.0.0"

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
    startupConf, err := os.ReadFile(w.ActivePath)
    if err != nil {
        logger.Fatal("failed to read environment file\n\tFile is either deleted or something very serious is wrong.\n\tHow could we manage to read before?")
    }

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
                //Wait a tiny bit for the OS to finish the file swap
                time.Sleep(100 * time.Millisecond)
                watcher.Add(w.ActivePath) // Helps prevent dangling reference

                if err = godotenv.Overload(w.ActivePath); err != nil {
                    logger.Error("Detected a change in", w.ActivePath+".", "\nDue to errors, changes to environment will not be applied.\n\t", err)
                    continue
                }

                if err = util.EnsureENV(); err != nil {
                    envMap, err := godotenv.Parse(bytes.NewReader(startupConf))
                    if err != nil {
                        logger.Fatal("BUG Encountered, logically this should not have happened but it still did.")
                    }

                    for key, value := range envMap {
                        os.Setenv(key, value)
                    }
                } else {
                    logger.Okay("Detected a change in environment file, successfully updated the configuration")
                }
            }
        }
    }
}

func main() {
    err := util.EnsureENV()
    if err != nil {
        logger.Fatal(err)
    }
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

