package main

import (
    "flag"
    "fmt"
    "os"
    "runtime"
    "strconv"
    "strings"

    "github.com/jelius-sama/logger"
)

type Flag uint8

const (
    FlagV Flag = iota
    FlagP
    FlagEnv
    FlagOut
    FlagIn
)

func (f Flag) Short() string {
    switch f {
    case FlagV:
        return "v"
    case FlagP:
        return "p"
    case FlagEnv:
        return "env"
    case FlagOut:
        return "o"
    case FlagIn:
        return "i"
    }
    logger.Panic("unreachable: f.Short()")
    return ""
}

func (f Flag) Long() string {
    switch f {
    case FlagV:
        return "version"
    case FlagP:
        return "port"
    case FlagEnv:
        return "env"
    case FlagOut:
        return "o"
    case FlagIn:
        return "i"
    }
    logger.Panic("unreachable: f.Long()")
    return ""
}

func (f Flag) String() string {
    if f.Long() != f.Short() {
        return "[-" + f.Long() + "|" + "-" + f.Short() + "]"
    } else {
        return "-" + f.Long()
    }
}

func handleFlags() bool {
    if len(os.Args) > 1 && os.Args[1] == "gen" {
        handleGenCmd(flag.NewFlagSet("gen", flag.ExitOnError))
        return true
    }

    if len(os.Args) > 1 && os.Args[1] == "cloudfront" {
        handleCloudfrontCmd(flag.NewFlagSet("cloudfront", flag.ExitOnError))
        return true
    }

    flag.Usage = func() {
        w := flag.CommandLine.Output()
        GlobalHelp(w)
        fmt.Fprintf(w, "\nFor more information, visit https://github.com/jelius-sama/OpenMediaCloud\n")
    }

    defaultPort, err := strconv.Atoi(strings.TrimPrefix(PORT, ":"))
    if err != nil {
        logger.Debug("Failed to get default server port:", err)
        logger.Debug("Setting :8000 as default port")
        defaultPort = 8000
    }

    CustomEnvPath = flag.String(FlagEnv.Long(), "", "Load environment variables from a custom path")

    flag.Bool(FlagV.Long(), false, "Show version and build info")
    flag.Bool(FlagV.Short(), false, "Show version and build info")

    customPort := flag.Int(FlagP.Long(), defaultPort, "Specify a custom port to run the proxy server on")
    customPort = flag.Int(FlagP.Short(), defaultPort, "Specify a custom port to run the proxy server on")

    flag.Parse()

    isFlagPassed := func(fg Flag) bool {
        found := false
        flag.Visit(func(f *flag.Flag) {
            if f.Name == fg.Long() || f.Name == fg.Short() {
                found = true
            }
        })
        return found
    }

    if isFlagPassed(FlagV) && (isFlagPassed(FlagEnv) || isFlagPassed(FlagP)) {
        logger.Fatal("\r"+"Flags", FlagEnv.String()+", "+FlagP.String()+", and "+FlagV.String(), "are mutually exclusive. Please use only one.")
    }

    if isFlagPassed(FlagV) {
        logger.Info("\r"+"OpenMediaCloud", VERSION, "\nCompiled for", runtime.GOOS, runtime.GOARCH)
        return true
    }

    if isFlagPassed(FlagP) {
        PORT = fmt.Sprintf(":%d", *customPort)
    }

    return false
}

func handleGenCmd(set *flag.FlagSet) {
    set.Usage = func() {
        w := flag.CommandLine.Output()
        fmt.Fprintf(w, "Usage: OpenMediaCloud gen [env|service] [options]\n\n")
        fmt.Fprintf(w, "Options for 'gen':\n")
        set.PrintDefaults()
        fmt.Fprintf(w, "\nExample:\n  OpenMediaCloud gen service -o /etc/systemd/system/OpenMediaCloud.service\n")
        fmt.Fprintf(w, "\nFor more information, visit https://github.com/jelius-sama/OpenMediaCloud\n")
    }
    genOut := set.String(FlagOut.Long(), "", "Output file path (default: stdout)")

    // Check for the second-level command (env or service)
    if len(os.Args) < 3 {
        logger.Fatal("\r" + "expected 'env' or 'service' after 'gen'")
    }

    subCommand := os.Args[2]

    // Parse flags starting from the 2nd argument
    // skips 'OpenMediaCloud', 'gen', 'env'|'service'
    set.Parse(os.Args[3:])

    var content string

    switch subCommand {
    case "env":
        content = ENV
    case "service":
        content = SERVICE
    case "-h", "--h", "-help", "--help":
        set.Usage()
    default:
        set.Usage()
        logger.Fatal("\r"+"Unknown generation target:", subCommand)
    }

    if genOut != nil && len(*genOut) != 0 {
        if err := os.WriteFile(*genOut, []byte(content), 0644); err != nil {
            logger.Error("\r"+"Error writing to file:", err.Error())
            logger.Info("\r" + content)
        } else {
            logger.Okay("\r"+"Successfully generated", subCommand, "to", *genOut)
        }
    } else {
        logger.Okay("\r" + content)
    }
}

func handleCloudfrontCmd(set *flag.FlagSet) {
    set.Usage = func() {
        w := flag.CommandLine.Output()
        fmt.Fprintf(w, "Usage: OpenMediaCloud cloudfront [cp <path> <path>|ls <path>]\n\n")
        fmt.Fprintf(w, "\nExample:\n  OpenMediaCloud cloudfront ls /Anime/ \n")
        fmt.Fprintf(w, "\nExample:\n  OpenMediaCloud cloudfront cp /Anime/* ~/Downloads/ \n")
        fmt.Fprintf(w, "\nFor more information, visit https://github.com/jelius-sama/OpenMediaCloud\n")
    }

    subCommand := os.Args[2]

    switch subCommand {
    case "ls":
        if len(os.Args) < 4 {
            logger.Fatal("\r" + "expected <path> after 'ls'")
        }

        items := decodePath(os.Args[3], 100)
        for i := len(items) - 1; i >= 0; i-- {
            logger.Okay("\r", items[i])
        }
    case "cp":
        if len(os.Args) < 5 {
            logger.Fatal("\r" + "expected <input path> and <output path> after 'cp'")
        }
        logger.Debug("TODO: Implement cloudfront cp command.")
    case "-h", "--h", "-help", "--help":
        set.Usage()
    default:
        set.Usage()
        logger.Fatal("\r"+"Unknown command:", subCommand)
    }
}

