package collectors

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/StackExchange/tcollector/opentsdb"
)

var collectors []Collector

const DEFAULT_FREQ_SEC = 15

type Collector struct {
	F func() opentsdb.MultiDataPoint
	seconds time.Duration
}

//type Collector func() opentsdb.MultiDataPoint

var l = log.New(os.Stdout, "", log.LstdFlags)

var host = "unknown"
var timestamp int64 = time.Now().Unix()

func init() {
	if h, err := os.Hostname(); err == nil {
		host = h
	}
	go func() {
		for t := range time.Tick(time.Second) {
			timestamp = t.Unix()
		}
	}()
}

// Search returns all collectors matching the pattern s.
func Search(s string) []Collector {
	var r []Collector
	for _, c := range collectors {
		v := runtime.FuncForPC(reflect.ValueOf(c.F).Pointer())
		if strings.Contains(v.Name(), s) {
			r = append(r, c)
		}
	}
	return r
}

func Run() chan *opentsdb.DataPoint {
	dpchan := make(chan *opentsdb.DataPoint)
	for _, c := range collectors {
		go runCollector(dpchan, c)
	}
	return dpchan
}

func runCollector(dpchan chan *opentsdb.DataPoint, c Collector) {
	for _ = range time.Tick(time.Second * c.seconds) {
		md := c.F()
		for _, dp := range md {
			dpchan <- dp
		}
	}
}

func Add(md *opentsdb.MultiDataPoint, name string, value interface{}, tags opentsdb.TagSet) {
	if tags == nil {
		tags = make(opentsdb.TagSet)
	}
	tags["host"] = host
	d := opentsdb.DataPoint{
		Metric:    name,
		Timestamp: timestamp,
		Value:     value,
		Tags:      tags,
	}
	*md = append(*md, &d)
}

func CreateQuery(t interface{}, where string) string {
	var b bytes.Buffer
	b.WriteString("SELECT ")

	s := reflect.ValueOf(t).Elem()

	typeOfT := s.Type()
	//Since we generally pass slices, this function takes the underlying or contained type of the slice
	ContainedtypeOfT := typeOfT.Elem()

	for i := 0; i < ContainedtypeOfT.NumField()-1; i++ {
		b.WriteString(fmt.Sprintf("%s, ", ContainedtypeOfT.Field(i).Name))
	}

	//Last one has no Comma
	b.WriteString(fmt.Sprintf("%s ", ContainedtypeOfT.Field(ContainedtypeOfT.NumField()-1).Name))

	b.WriteString(fmt.Sprintf("FROM %s ", ContainedtypeOfT.Name()))
	b.WriteString(where)
	return (b.String())
}

func readProc(fname string, line func(string)) {
	f, err := os.Open(fname)
	if err != nil {
		l.Printf("%v: %v\n", fname, err)
		return
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		l.Printf("%v: %v\n", fname, err)
	}
}

// IsDigit returns true if s consists of decimal digits.
func IsDigit(s string) bool {
	r := strings.NewReader(s)
	for {
		ch, _, err := r.ReadRune()
		if ch == 0 || err != nil {
			break
		} else if ch == utf8.RuneError {
			return false
		} else if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

// IsAlNum returns true if s is alphanumeric.
func IsAlNum(s string) bool {
	r := strings.NewReader(s)
	for {
		ch, _, err := r.ReadRune()
		if ch == 0 || err != nil {
			break
		} else if ch == utf8.RuneError {
			return false
		} else if !unicode.IsDigit(ch) && !unicode.IsLetter(ch) {
			return false
		}
	}
	return true
}
