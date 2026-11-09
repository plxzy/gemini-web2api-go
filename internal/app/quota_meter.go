package app

// Local patch: GET /v1/quota returns the raw qpEbW (CheckGeminiQuota) payloads per
// enabled account, so a caller can snapshot the compute meter before/after a call.
// Layout is not yet decoded; raw payloads are returned for validation.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func qpEbW(a CookieAccount, args string) (any, error) {
	sapisid := extractSAPISID(a.Cookie)
	var out any
	err := viaSlot(a, func(proxyURL string) error {
		xsrf, err := getXSRF(a.Cookie, a.AuthUser, proxyURL)
		if err != nil {
			return err
		}
		freq, _ := json.Marshal([]any{[]any{[]any{"qpEbW", args, nil, "generic"}}})
		form := url.Values{"f.req": {string(freq)}, "at": {xsrf}}
		ep := fmt.Sprintf("%s/_/BardChatUi/data/batchexecute?rpcids=qpEbW&bl=%s&hl=en&_reqid=%d&rt=c",
			geminiBase(a.AuthUser), currentBL(proxyURL), time.Now().UnixNano()%1000000)
		h := buildGeminiHeaders(a.Cookie, sapisid, "")
		setAuthUserHeaders(h, a.AuthUser)
		delete(h, "x-goog-ext-525001261-jspb")
		st, _, body, err := uploadPost(ep, h, []byte(form.Encode()), proxyURL)
		if err != nil {
			return err
		}
		if st != 200 {
			return fmt.Errorf("qpEbW HTTP %d", st)
		}
		out = wrbPayloads(string(body))
		return nil
	})
	return out, err
}

func handleQuota(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query().Get("args")
	if args == "" {
		args = "[[[1,2]]]"
	}
	var accts []map[string]any
	for _, a := range accountList() {
		if a.Status != "enabled" {
			continue
		}
		row := map[string]any{"id": a.ID, "label": accountDisplayName(&a)}
		if p, err := qpEbW(a, args); err != nil {
			row["error"] = err.Error()
		} else {
			row["payloads"] = p
		}
		accts = append(accts, row)
	}
	writeJSON(w, 200, map[string]any{"now": time.Now().Unix(), "args": args, "accounts": accts})
}
