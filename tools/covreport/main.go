package main

// Command covreport generates a self-contained HTML coverage report from a Go
// coverage profile. It includes a consolidated summary table (per-package),
// a detailed per-file breakdown, and annotated source code with line-level
// coverage highlighting — all in a single HTML file.
//
// Usage: go run ./tools/covreport -i coverage.out -o coverage.html

import (
	"bufio"
	"flag"
	"fmt"
	"html"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// coverBlock is a single coverage block from the profile.
type coverBlock struct {
	StartLine int
	EndLine   int
	NumStmts  int
	Count     int
}

// profileFile holds raw profile data for one source file.
type profileFile struct {
	blocks  []coverBlock
	stmts   int
	covered int
}

// fileDetail holds coverage info and annotated source for one file.
type fileDetail struct {
	Name     string // base filename
	Stmts    int
	Covered  int
	Coverage float64
	Source   template.HTML // pre-rendered annotated HTML
}

// pkgSummary holds aggregate coverage for one package.
type pkgSummary struct {
	Package  string
	Stmts    int
	Covered  int
	Coverage float64
	Files    []fileDetail
}

// reportData is the top-level template data.
type reportData struct {
	Total    float64
	Packages []pkgSummary
}

func main() {
	inFile := flag.String("i", "coverage.out", "input coverage profile")
	outFile := flag.String("o", "coverage.html", "output HTML file")
	flag.Parse()

	modRoot, modPath, err := findModule()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot find module root: %v (source annotations disabled)\n", err)
	}

	files, err := parseProfile(*inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	data := buildReport(files, modRoot, modPath)

	f, err := os.Create(*outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := reportTmpl.Execute(f, data); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Coverage report: %s (%.1f%%)\n", *outFile, data.Total)
}

// findModule reads go.mod in the current directory to determine the module
// path and root directory.
func findModule() (root string, modPath string, err error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modPath = strings.TrimSpace(line[7:])
			break
		}
	}
	if modPath == "" {
		return "", "", fmt.Errorf("module directive not found in go.mod")
	}
	root, err = os.Getwd()
	return root, modPath, err
}

// resolveFile maps an import path from the profile to a local filesystem path.
func resolveFile(importPath, modRoot, modPath string) string {
	if modRoot == "" || modPath == "" {
		return ""
	}
	if !strings.HasPrefix(importPath, modPath) {
		return ""
	}
	rel := importPath[len(modPath):]
	if strings.HasPrefix(rel, "/") {
		rel = rel[1:]
	}
	return filepath.Join(modRoot, rel)
}

// parseProfile reads a coverage profile and returns per-file block data.
func parseProfile(profilePath string) (map[string]*profileFile, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	files := make(map[string]*profileFile)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		// Format: file:startLine.startCol,endLine.endCol numStmts count
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		numStmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		filePart := parts[0]
		colonIdx := strings.IndexByte(filePart, ':')
		if colonIdx < 0 {
			continue
		}
		filePath := filePart[:colonIdx]
		posRange := filePart[colonIdx+1:]

		startLine, endLine := parseRange(posRange)

		pf, ok := files[filePath]
		if !ok {
			pf = &profileFile{}
			files[filePath] = pf
		}
		pf.blocks = append(pf.blocks, coverBlock{
			StartLine: startLine,
			EndLine:   endLine,
			NumStmts:  numStmts,
			Count:     count,
		})
		pf.stmts += numStmts
		if count > 0 {
			pf.covered += numStmts
		}
	}
	return files, scanner.Err()
}

// parseRange extracts startLine and endLine from "startLine.startCol,endLine.endCol".
func parseRange(s string) (int, int) {
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return 0, 0
	}
	startPart := s[:comma]
	endPart := s[comma+1:]
	startLine := 0
	endLine := 0
	if dot := strings.IndexByte(startPart, '.'); dot >= 0 {
		startLine, _ = strconv.Atoi(startPart[:dot])
	}
	if dot := strings.IndexByte(endPart, '.'); dot >= 0 {
		endLine, _ = strconv.Atoi(endPart[:dot])
	}
	return startLine, endLine
}

// annotateSource reads a source file and produces line-by-line annotated HTML.
func annotateSource(localPath string, blocks []coverBlock) template.HTML {
	if localPath == "" {
		return template.HTML("<em class=\"src-na\">source not available</em>")
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return template.HTML("<em class=\"src-na\">source not available: " +
			html.EscapeString(err.Error()) + "</em>")
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line from final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Build per-line status: 0 = not instrumented, 1 = covered, -1 = not covered.
	status := make([]int, len(lines))
	for _, b := range blocks {
		for ln := b.StartLine; ln <= b.EndLine && ln <= len(lines); ln++ {
			idx := ln - 1
			if idx < 0 {
				continue
			}
			if b.Count > 0 {
				status[idx] = 1
			} else if status[idx] == 0 {
				status[idx] = -1
			}
		}
	}

	var buf strings.Builder
	maxDigits := len(strconv.Itoa(len(lines)))
	fmtStr := fmt.Sprintf("%%%dd", maxDigits)
	for i, line := range lines {
		cls := "neu"
		switch status[i] {
		case 1:
			cls = "cov"
		case -1:
			cls = "uncov"
		}
		lineNum := fmt.Sprintf(fmtStr, i+1)
		escaped := html.EscapeString(line)
		fmt.Fprintf(&buf, "<span class=\"%s\"><span class=\"ln\">%s</span>  %s</span>\n",
			cls, lineNum, escaped)
	}
	return template.HTML(buf.String())
}

func buildReport(files map[string]*profileFile, modRoot, modPath string) reportData {
	// Group files by package.
	pkgMap := make(map[string][]string) // pkg → []filePath
	for filePath := range files {
		pkg := path.Dir(filePath)
		pkgMap[pkg] = append(pkgMap[pkg], filePath)
	}

	var packages []pkgSummary
	totalStmts := 0
	totalCovered := 0

	for pkg, filePaths := range pkgMap {
		sort.Strings(filePaths)
		ps := pkgSummary{Package: pkg}

		for _, fp := range filePaths {
			pf := files[fp]
			ps.Stmts += pf.stmts
			ps.Covered += pf.covered

			cov := 0.0
			if pf.stmts > 0 {
				cov = float64(pf.covered) / float64(pf.stmts) * 100
			}

			localPath := resolveFile(fp, modRoot, modPath)
			ps.Files = append(ps.Files, fileDetail{
				Name:     path.Base(fp),
				Stmts:    pf.stmts,
				Covered:  pf.covered,
				Coverage: cov,
				Source:   annotateSource(localPath, pf.blocks),
			})
		}

		if ps.Stmts > 0 {
			ps.Coverage = float64(ps.Covered) / float64(ps.Stmts) * 100
		}
		totalStmts += ps.Stmts
		totalCovered += ps.Covered
		packages = append(packages, ps)
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Package < packages[j].Package
	})

	total := 0.0
	if totalStmts > 0 {
		total = float64(totalCovered) / float64(totalStmts) * 100
	}
	return reportData{Total: total, Packages: packages}
}

var reportTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"covClass": func(cov float64) string {
		switch {
		case cov >= 80:
			return "high"
		case cov >= 50:
			return "med"
		default:
			return "low"
		}
	},
	"fmtCov": func(cov float64) string {
		return fmt.Sprintf("%.1f%%", cov)
	},
	"shortPkg": func(pkg string) string {
		const prefix = "github.com/diptopandit/tork/"
		if strings.HasPrefix(pkg, prefix) {
			return pkg[len(prefix):]
		}
		return pkg
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>tork — Coverage Report</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         background: #0d1117; color: #c9d1d9; padding: 2rem; }
  h1 { font-size: 1.5rem; margin-bottom: 0.25rem; }
  h2 { font-size: 1.1rem; margin-bottom: 0.8rem; }
  .subtitle { color: #8b949e; margin-bottom: 1.5rem; }
  .total-badge { display: inline-block; font-size: 2rem; font-weight: bold; padding: 0.5rem 1.5rem;
                 border-radius: 8px; margin-bottom: 1.5rem; }
  .total-badge.high { background: #1a4731; color: #3fb950; }
  .total-badge.med  { background: #3d2e00; color: #d29922; }
  .total-badge.low  { background: #4a1d1d; color: #f85149; }

  table { width: 100%; border-collapse: collapse; margin-bottom: 1.5rem; }
  th { text-align: left; padding: 0.6rem 0.8rem; border-bottom: 2px solid #30363d;
       color: #8b949e; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; }
  td { padding: 0.5rem 0.8rem; border-bottom: 1px solid #21262d; }
  tr:hover { background: #161b22; }

  .bar-cell { width: 200px; }
  .bar-bg { background: #21262d; border-radius: 4px; height: 8px; position: relative; }
  .bar-fill { height: 8px; border-radius: 4px; position: absolute; top: 0; left: 0; }
  .bar-fill.high { background: #3fb950; }
  .bar-fill.med  { background: #d29922; }
  .bar-fill.low  { background: #f85149; }

  .cov-num { font-variant-numeric: tabular-nums; min-width: 4rem; text-align: right; }
  .cov-num.high { color: #3fb950; }
  .cov-num.med  { color: #d29922; }
  .cov-num.low  { color: #f85149; }

  .pkg-name { font-family: "SF Mono", Menlo, monospace; font-size: 0.9rem; }
  .stmts { color: #8b949e; font-size: 0.85rem; }

  /* package-level accordion */
  details.pkg-detail { margin-bottom: 0.5rem; }
  details.pkg-detail > summary { cursor: pointer; padding: 0.4rem 0; }
  details.pkg-detail > summary:hover { color: #58a6ff; }

  /* file-level accordion inside package */
  details.file-detail { margin: 0.25rem 0 0.25rem 1.5rem; }
  details.file-detail > summary {
    cursor: pointer; padding: 0.3rem 0; font-size: 0.88rem;
    display: flex; align-items: center; gap: 0.8rem;
  }
  details.file-detail > summary:hover { color: #58a6ff; }
  .file-bar { width: 120px; display: inline-block; }
  .file-cov { min-width: 3.5rem; text-align: right; font-size: 0.85rem; }
  .file-stmts { color: #8b949e; font-size: 0.8rem; min-width: 4rem; }

  /* annotated source code */
  pre.source {
    background: #161b22; border: 1px solid #30363d; border-radius: 6px;
    padding: 0.8rem 1rem; overflow-x: auto; margin: 0.5rem 0 0.5rem 0;
    font-family: "SF Mono", Menlo, Consolas, monospace; font-size: 0.78rem; line-height: 1.1;
  }
  pre.source > span { display: block; white-space: pre; }
  pre.source .ln { display: inline; color: #484f58; user-select: none; }
  pre.source .cov { color: #3fb950; }
  pre.source .uncov { color: #f85149; background: rgba(248,81,73,0.08); }
  pre.source .neu { color: #6e7681; }
  .src-na { color: #8b949e; font-style: italic; display: block; padding: 0.5rem 1rem; }
</style>
</head>
<body>
<h1>tork — Coverage Report</h1>
<p class="subtitle">Generated by <code>make test-coverage-html</code></p>

<div class="total-badge {{covClass .Total}}">{{fmtCov .Total}}</div>

<h2>Package Summary</h2>
<table>
<thead>
<tr><th>Package</th><th>Stmts</th><th>Covered</th><th class="bar-cell">Coverage</th><th></th></tr>
</thead>
<tbody>
{{range .Packages}}
<tr>
  <td class="pkg-name">{{shortPkg .Package}}</td>
  <td class="stmts">{{.Stmts}}</td>
  <td class="stmts">{{.Covered}}</td>
  <td class="bar-cell"><div class="bar-bg"><div class="bar-fill {{covClass .Coverage}}" style="width:{{fmtCov .Coverage}}"></div></div></td>
  <td class="cov-num {{covClass .Coverage}}">{{fmtCov .Coverage}}</td>
</tr>
{{end}}
</tbody>
</table>

<h2>File Detail</h2>
{{range .Packages}}
<details class="pkg-detail">
<summary class="pkg-name">{{shortPkg .Package}} — <span class="cov-num {{covClass .Coverage}}">{{fmtCov .Coverage}}</span></summary>
{{range .Files}}
<details class="file-detail">
  <summary>
    <span class="pkg-name">{{.Name}}</span>
    <span class="file-stmts">{{.Covered}}/{{.Stmts}}</span>
    <span class="file-bar"><div class="bar-bg"><div class="bar-fill {{covClass .Coverage}}" style="width:{{fmtCov .Coverage}}"></div></div></span>
    <span class="file-cov cov-num {{covClass .Coverage}}">{{fmtCov .Coverage}}</span>
  </summary>
  <pre class="source">{{.Source}}</pre>
</details>
{{end}}
</details>
{{end}}

</body>
</html>
`))
