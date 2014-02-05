package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type HTMLContent struct {
	Content string
}

var (
	layoutFilename = "_layout.html"
	srcDirname     = "_src"
)

func main() {
	httpFlag := flag.String("http", "", "addr. (e.g. \":8000\")")
	flag.Parse()

	var err error

	if *httpFlag != "" {
		err = serve(*httpFlag)
	} else {
		err = build()
	}

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}

func serve(addr string) error {
	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	log.Printf("Starging web server: %s", addr)
	err = http.ListenAndServe(addr, http.FileServer(http.Dir(wd)))

	if err != nil {
		return err
	}

	return nil
}

func build() error {
	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	srcPath := filepath.Join(wd, srcDirname)

	if _, err := os.Stat(srcPath); err != nil {
		return err
	}

	layoutFilepath := filepath.Join(srcPath, layoutFilename)

	if _, err := os.Stat(layoutFilepath); err != nil {
		return err
	}

	err = filepath.Walk(srcPath, func(inputFilepath string, fileInfo os.FileInfo, err error) error {
		// skip directory
		if fileInfo.IsDir() {
			return nil
		}

		// skip layout file
		if inputFilepath == layoutFilepath {
			return nil
		}

		// run GOG_BUILD.go
		if strings.HasSuffix(inputFilepath, "/GOG_BUILD.go") {
			return goRun(inputFilepath)
		}

		// create output file
		outputFilepath := strings.Replace(inputFilepath, srcPath, wd, 1)

		err = mkdirIfNotExist(filepath.Dir(outputFilepath))

		if err != nil {
			return err
		}

		out, err := os.Create(outputFilepath)

		if err != nil {
			return err
		}

		defer out.Close()

		// open input file
		in, err := os.Open(inputFilepath)

		if err != nil {
			return err
		}

		defer in.Close()

		// build
		switch filepath.Ext(in.Name()) {
		case ".html":
			content, err := ioutil.ReadFile(in.Name())

			if err != nil {
				return err
			}

			w := bufio.NewWriter(out)
			t, err := template.ParseFiles(layoutFilepath)

			if err != nil {
				return err
			}

			t.Execute(w, &HTMLContent{string(content)})
			w.Flush()
		default:
			_, err = io.Copy(out, in)

			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func mkdirIfNotExist(dir string) error {
	_, err := os.Stat(dir)

	if err != nil {
		err := os.MkdirAll(dir, os.ModeDir|0755)

		if err != nil {
			return err
		}
	}

	return nil
}

func goRun(path string) error {
	cmd := exec.Command("go")

	cmd.Args = []string{"go", "run", path}
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
