package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	//ht "html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	tt "text/template"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	s := createSettingsFromFlags()

	if err := s.validate(); err != nil {
		log.Fatal(err)
	}

	if err := s.parseCodeTemplates(); err != nil {
		log.Fatal(err)
	}

	if err := s.splice(); err != nil {
		log.Fatal(err)
	}
}

type uuid struct {
	UUId string
}

type settings struct {
	htmlPath           string
	codePath           string
	outDir             string
	htmlInputs         []string
	codeInputs         []string
	codeExcerpts       *tt.Template
	emptyCommentRegexp *regexp.Regexp
}

func createSettingsFromFlags() *settings {
	var htmlPath, codePath, outDir string
	flag.StringVar(&htmlPath, "html", "", "`Path` to the html sources (dir or file)")
	flag.StringVar(&codePath, "code", "", "`Path` to the code sources (dir or file)")
	flag.StringVar(&outDir, "out", "", "`Path` to the output (dir only)")
	flag.Parse()

	return &settings{
		htmlPath:           htmlPath,
		codePath:           codePath,
		outDir:             outDir,
		emptyCommentRegexp: regexp.MustCompile(`(?m:^//\s*[\n\r]*)`),
	}
}

func (s *settings) splice() error {
	for _, inPath := range s.htmlInputs {
		log.Printf("Processing %s", inPath)
		content, err := ioutil.ReadFile(inPath)
		if err != nil {
			return err
		}
		tpl, err := s.codeExcerpts.Clone()
		if err != nil {
			return err
		}
		tpl, err = tpl.Parse(string(content))
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		if tpl.Execute(buf, nil); err != nil {
			return err
		}

		rel, err := filepath.Rel(s.htmlPath, inPath)
		if err != nil {
			return err
		}
		outPath := filepath.Join(s.outDir, rel)
		if err = os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		if err = ioutil.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *settings) pygmentize(lang string, content string) (string, error) {
	cmd := exec.Command("pygmentize", "-f", "html", "-l", lang, "-P", "style=autumn")
	cmd.Stdin = strings.NewReader(content)
	out := new(bytes.Buffer)
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return string(out.Bytes()), nil
}

func (s *settings) parseCodeTemplates() error {
	excerpts := make(map[string]string, len(s.codeInputs))

	for _, pathIn := range s.codeInputs {
		codeTemplate, err := tt.ParseFiles(pathIn)
		if err != nil {
			return err
		}
		lang := (filepath.Ext(pathIn))[1:]
		for _, tpl := range codeTemplate.Templates() {
			if tpl.Name() == filepath.Base(pathIn) {
				continue
			}
			log.Printf("In %s (%s) found %s", pathIn, lang, tpl.Name())
			buf := new(bytes.Buffer)
			err := tpl.Execute(buf, nil)
			if err != nil {
				return err
			}
			str := string(s.emptyCommentRegexp.ReplaceAll(buf.Bytes(), nil))
			str, err = s.pygmentize(lang, str)
			if err != nil {
				return err
			}
			excerpts[tpl.Name()] = str
		}
	}

	codeExcerpts := tt.New("excerpts")
	for uuid, content := range excerpts {
		_, err := codeExcerpts.Parse(fmt.Sprintf("{{define %q}}%s{{end}}", uuid, content))
		if err != nil {
			return err
		}
	}
	s.codeExcerpts = codeExcerpts

	return nil
}

func (s *settings) validate() error {
	if len(s.htmlPath) == 0 {
		return errors.New("No html path supplied")
	}
	htmlPath, htmlInputs, err := find(s.htmlPath, ".html")
	if err != nil {
		return err
	}
	s.htmlPath, s.htmlInputs = htmlPath, htmlInputs

	if len(s.codePath) == 0 {
		return errors.New("No code path supplied")
	}
	codePath, codeInputs, err := find(s.codePath)
	if err != nil {
		return err
	}
	s.codePath, s.codeInputs = codePath, codeInputs

	return os.MkdirAll(s.outDir, 0755)
}

func find(path string, extensions ...string) (string, []string, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}

	if stat, err := os.Stat(path); err != nil {
		return "", nil, err
	} else if mode := stat.Mode(); mode.IsDir() {
		results := make([]string, 0, 16)
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				if len(extensions) != 0 {
					ext := filepath.Ext(p)
					found := false
					for _, e := range extensions {
						if found = e == ext; found {
							break
						}
					}
					if !found {
						return nil
					}
				}
				results = append(results, p)
			}
			return nil
		})
		if err != nil {
			return "", nil, err
		}
		return path, results, nil
	} else if mode.IsRegular() {
		return filepath.Dir(path), []string{path}, nil
	} else {
		return "", nil, fmt.Errorf("%s appears to be neither a directory or a file.", path)
	}
}
