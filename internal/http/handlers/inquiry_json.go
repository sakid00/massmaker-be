package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/sakid00/massmaker-be/internal/service"
)

func decodeInquiry(r *http.Request, dst *service.CreateInquiryInput) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	cleaned, err := coerceInquiryJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(cleaned, dst)
}

func coerceInquiryJSON(raw []byte) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	for _, key := range []string{
		"quantity", "designCount", "sampleQuantity", "budgetMinIdr", "budgetMaxIdr",
	} {
		if v, ok := top[key]; ok {
			top[key] = coerceJSONInt(v)
		}
	}
	if atts, ok := top["attachments"]; ok {
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(atts, &rows); err == nil {
			for i := range rows {
				if v, ok := rows[i]["byteSize"]; ok {
					rows[i]["byteSize"] = coerceJSONInt(v)
				}
			}
			encoded, err := json.Marshal(rows)
			if err != nil {
				return nil, err
			}
			top["attachments"] = encoded
		}
	}
	return json.Marshal(top)
}

func coerceJSONInt(v json.RawMessage) json.RawMessage {
	s := strings.TrimSpace(string(v))
	if s == "" || s == "null" || s == `""` {
		return json.RawMessage("null")
	}
	var asString string
	if err := json.Unmarshal(v, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return json.RawMessage("null")
		}
		return numberOrNull(asString)
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return json.RawMessage("null")
	}
	return json.RawMessage(strconv.FormatInt(int64(math.Trunc(f)), 10))
}

func numberOrNull(s string) json.RawMessage {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return json.RawMessage("null")
	}
	return json.RawMessage(strconv.FormatInt(int64(math.Trunc(f)), 10))
}

func decodeInquiryFrom(raw []byte, dst *service.CreateInquiryInput) error {
	cleaned, err := coerceInquiryJSON(raw)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(cleaned))
	return dec.Decode(dst)
}
