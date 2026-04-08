package main

import (
    "flag"
    "fmt"
    "os"
    "runtime"

    "github.com/jelius-sama/logger"
)

func handleFlags() bool {
    if len(os.Args) > 1 && os.Args[1] == "gen" {
        handleGenCmd(flag.NewFlagSet("gen", flag.ExitOnError))
        return true
    }

    flag.Usage = func() {
        w := flag.CommandLine.Output()
        GlobalHelp(w)
        // flag.PrintDefaults()
        fmt.Fprintf(w, "\nFor more information, visit https://github.com/jelius-sama/OpenMediaCloud\n")
    }

    CustomEnvPath = flag.String("env", "", "Load environment variables from a custom path")
    showVersion := flag.Bool("version", false, "Show version and build info")

    flag.Parse()

    if flag.NFlag() > 1 {
        logger.Fatal("\r" + "Flags -env, and -version are mutually exclusive. Please use only one.")
    }

    if showVersion != nil && *showVersion == true {
        logger.Info("\r"+"OpenMediaCloud", VERSION, "\nCompiled for", runtime.GOOS, runtime.GOARCH)
        return true
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
    genOut := set.String("o", "", "Output file path (default: stdout)")

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
        logger.Fatal("\r"+"Unknown generation target:", subCommand)
    }

    if genOut != nil && len(*genOut) != 0 {
        err := os.WriteFile(*genOut, []byte(content), 0644)
        if err != nil {
            logger.Error("\r"+"Error writing to file:", err.Error())
            logger.Info("\r" + content)
        } else {
            logger.Okay("\r"+"Successfully generated", subCommand, "to", *genOut)
        }
    } else {
        logger.Okay("\r" + content)
    }
}

