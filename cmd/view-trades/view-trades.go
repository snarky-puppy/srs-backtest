package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/mwlazlo/srs/internal/exchange"
)

type ReportReader struct {
	baseDir string
	files   []string
}

type position struct {
	prev string
	cur  string
	next string
}

func (r *ReportReader) Report(cursor string) position {
	idx := sort.SearchStrings(r.files, cursor)
	if idx == len(r.files) {
		return r.Report(r.files[0])
	}
	if idx == 0 {
		return position{
			prev: "",
			cur:  r.files[0],
			next: r.files[1],
		}
	}
	if idx == len(r.files)-1 {
		return position{
			prev: r.files[idx-1],
			cur:  r.files[idx],
			next: "",
		}
	}
	return position{
		prev: r.files[idx-1],
		cur:  r.files[idx],
		next: r.files[idx+1],
	}
}

func (r *ReportReader) LoadRecord(cur string) (rv exchange.HistoricalRecord) {
	fp, err := os.Open(fmt.Sprintf("%s/%s", r.baseDir, cur))
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	err = json.NewDecoder(fp).Decode(&rv)
	if err != nil {
		panic(err)
	}
	return
}

// LoadAll loads all the reports in the base directory
func (r *ReportReader) LoadAll() (rv []exchange.HistoricalRecord) {
	for _, f := range r.files {
		rv = append(rv, r.LoadRecord(f))
	}
	return
}

func NewReportReader(baseDir string) *ReportReader {

	files, err := os.ReadDir(baseDir)
	if err != nil {
		panic(err)
	}

	var names []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			names = append(names, f.Name())
		}
	}

	sort.Strings(names)

	return &ReportReader{
		files:   names,
		baseDir: baseDir,
	}
}

func main() {

	fmt.Println("opening http://localhost:8081/")
	http.Handle("/static/", http.FileServer(http.Dir("data/reports")))
	http.HandleFunc("/", endpoint())
	_ = http.ListenAndServe("localhost:8081", nil)

}

func endpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the day from the query string
		rr := NewReportReader("data/reports")

		if r.URL.Path == "/daily" {
			data := rr.LoadAll()
			RenderDaily(w, data)
			return
		}

		pos := rr.Report(r.URL.Query().Get("d"))
		report := rr.LoadRecord(pos.cur)
		RenderPage(w, pos, report)
	}
}
