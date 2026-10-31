package main

import (
	"cmp"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

func env(k string) string { return os.Getenv("NANASHI_" + k) }

func newAddr() (s string) {
	for range 8 {
		n, _ := rand.Int(rand.Reader, big.NewInt(36))
		s += n.Text(36)
	}
	return s + "@" + env("DOMAIN")
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func appendAlias(addr, note string) error {
	f, err := os.OpenFile(cmp.Or(env("ALIASES"), "/etc/aliases"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	_, err = fmt.Fprintf(f, "# %s %s\n%s: %s\n", time.Now().Format(time.RFC3339), note, addr, env("TO"))
	return err
}

func newAlias(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("authentication")), []byte(env("KEY"))) != 1 {
		reply(w, 401, map[string]string{"error": "Wrong api key"})
		return
	}
	var req struct{ Note string }
	json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req)
	addr := newAddr()
	if err := appendAlias(addr, strings.Join(strings.Fields(req.Note), " ")); err != nil {
		reply(w, 500, map[string]string{"error": "Storage error"})
		return
	}
	reply(w, 201, map[string]string{"alias": addr, "email": addr})
}

func main() {
	flag.Parse()
	version := flag.Bool("version", false, "print the version")
	listen := flag.String("listen", "", "serve the API on this address")
	if *version {
		bi, _ := debug.ReadBuildInfo()
		fmt.Println(bi.Main.Version)
		return
	}
	if env("DOMAIN") == "" {
		log.Fatal("NANASHI_DOMAIN is required")
	}
	if *listen == "" {
		fmt.Println(newAddr())
		return
	}
	if env("KEY") == "" || env("TO") == "" {
		log.Fatal("NANASHI_KEY and NANASHI_TO are required")
	}
	http.HandleFunc("POST /api/alias/random/new", newAlias)
	srv := &http.Server{Addr: *listen, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
