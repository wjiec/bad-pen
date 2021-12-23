package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
)

var (
	healthy = -1
	ListenAddr = ":8080"
)

func init() {
	if addr := os.Getenv("SERVICE_LISTEN_ADDR"); addr != "" {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			log.Fatalf("invalid address %q to listen: %s", addr, err)
		}
		ListenAddr = addr
	}

	if count := os.Getenv("SERVICE_HEALTHY_COUNT"); count != "" {
		if num, err := strconv.Atoi(count); err == nil {
			healthy = num
		}
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("received request from %s (method=%s, uri=%s)", r.RemoteAddr, r.Method, r.RequestURI)
		if healthy > 0 {
			healthy--
		} else if healthy == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if hostname, err := os.Hostname(); err == nil {
			_, _ = w.Write([]byte(fmt.Sprintf("You've hit <%s> from %q\n\n", hostname, r.RemoteAddr)))
			SortHeaderMap(r.Header, func(k string, v []string) {
				_, _ = w.Write([]byte(fmt.Sprintf("%q => %q\n", k, v[0])))
			})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("error: %s", err)))
		}
	})

	log.Printf("service has been started on %q", ListenAddr)
	if err := http.ListenAndServe(ListenAddr, nil); err != nil {
		log.Printf("fatal error: %s", err)
	}
}

func SortHeaderMap(header http.Header, f func(k string, v []string)) {
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}

	sort.StringSlice(keys).Sort()
	for _, k := range keys {
		f(k, header[k])
	}
}
