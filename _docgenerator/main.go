package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	flagDoWrite    = flag.Bool("write", false, "Write to the opened file")
	flagClean      = flag.Bool("clean", false, "Removes the added documentation (use with -write)")
	flagLuaVersion = flag.String("luaversion", "5.1", "Lua documentation version")
	// flagHelp       = flag.Bool("help", false, "Show help")
	// flagHelpShort  = flag.Bool("h", false, "Show help")
)

func main() {
	flag.Parse()
	// if *flagHelp || *flagHelpShort {
	// 	flag.Usage()
	// 	return
	// }
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [-write] [-clean] [-luaversion <version | 5.1>] <go-file> ", os.Args[0])
		os.Exit(1)
	}

	goFile := flag.Arg(0)

	// Read Go code from file
	goCode, err := os.ReadFile(goFile)
	if err != nil {
		fmt.Println("Error reading Go file:", err)
		os.Exit(1)
	}

	updatedCode := ""
	if *flagClean {
		updatedCode = removeOldDocumentation(string(goCode))
	} else {

		// Fetch or load cached Lua documentation
		luadocs, err := getCachedLuaDocs()
		if err != nil {
			fmt.Println("Error fetching Lua documentation:", err)
			os.Exit(1)
		}

		luaFuncs, err := parseLuaDocs(luadocs)
		if err != nil {
			fmt.Println("Error parsing Lua documentation:", err)
			os.Exit(1)
		}

		// Add comments to the Go code
		updatedCode = addDocumentation(string(goCode), luaFuncs)
	}

	if *flagDoWrite {
		err := os.WriteFile(goFile, []byte(updatedCode), 0644)
		if err != nil {
			fmt.Println("Error writing annotated code to file:", err)
			os.Exit(1)
		}
		fmt.Println("Annotated code written back to", goFile)
	} else {
		fmt.Println(updatedCode)
	}
}

// Fetch or load cached Lua documentation
func getCachedLuaDocs() (string, error) {
	cacheDir := ".cache"
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("lua_manual_%s.html", *flagLuaVersion))

	// Ensure cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err := os.Mkdir(cacheDir, 0755)
		if err != nil {
			return "", err
		}
	}

	// Check if cached file exists
	var body string
	if _, err := os.Stat(cacheFile); err == nil {
		cachedBody, err := os.ReadFile(cacheFile)
		if err != nil {
			return "", err
		}
		body = string(cachedBody)
	} else {
		// Fetch Lua manual and cache it
		url := fmt.Sprintf("https://www.lua.org/manual/%s/manual.html", *flagLuaVersion) // Change version as needed
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return "", fmt.Errorf("unexpected error code for lua doc: %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		body = string(bodyBytes)

		err = os.WriteFile(cacheFile, bodyBytes, 0644)
		if err != nil {
			return "", err
		}
	}
	return body, nil
}

type LuaFunction struct {
	Name        string
	StackEffect string
	Signature   string
	Description string
}

func parseLuaDocs(html string) (map[string]LuaFunction, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	functions := map[string]LuaFunction{}

	doc.Find("hr + h3").Has("a[name^='lua_']").Each(func(i int, s *goquery.Selection) {
		var luaFunc LuaFunction

		// Extract name
		luaFunc.Name = s.Find("code").Text()

		// Extract stack effect
		if goquery.NodeName(s.Next()) == "p" {
			luaFunc.StackEffect = strings.TrimSpace(s.Next().Text())
			s = s.Next()
		}

		if goquery.NodeName(s.Next()) == "pre" {
			luaFunc.Signature = strings.TrimSpace(s.Next().Text())
			s = s.Next()
		}

		if goquery.NodeName(s.Next()) == "p" {
			luaFunc.Description = strings.ReplaceAll(strings.TrimSpace(s.Next().Text()), "\n", " ")
			// s = s.Next()
		}

		functions[strings.ToLower(luaFunc.Name)] = luaFunc
	})

	return functions, nil
}

func removeOldDocumentation(code string) string {
	blockCommentRegex := regexp.MustCompile(`(?m)(// lua_.*$\n)/\*\n(?: \*.*$\n)+[ ]?\*/$\n`)
	return blockCommentRegex.ReplaceAllString(code, "$1")
}

// Add documentation comments to the Go code
func addDocumentation(code string, luaFuncs map[string]LuaFunction) string {
	code = removeOldDocumentation(code)

	lines := strings.Split(code, "\n")
	var annotatedLines []string

	for _, line := range lines {
		annotatedLines = append(annotatedLines, line)
		if strings.HasPrefix(line, "// lua_") {
			functionName := strings.TrimPrefix(line, "// ")
			if doc, exists := luaFuncs[functionName]; exists {
				annotatedLines = append(annotatedLines, "/*")
				annotatedLines = append(annotatedLines, " * "+doc.StackEffect)
				annotatedLines = append(annotatedLines, " * "+doc.Description)
				annotatedLines = append(annotatedLines, " */")
			}
		}
	}

	return strings.Join(annotatedLines, "\n")
}
