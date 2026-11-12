package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// handleImagesGenerations 实现 OpenAI Images API：POST /v1/images/generations。
//
// 内部固定走 gemini-image（Nano Banana）——Gemini 网页端只有这一个生图后端，
// 所以请求里的 model 会被忽略。产物是 base64，默认返回 b64_json；
// response_format=url 时返回 data URL（本项目没有图床，不是 http 链接）。
// size / quality / style 同样忽略：分辨率由账号等级决定（免费 1K，付费 2K）。
func handleImagesGenerations(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeImageError(w, 400, "invalid_request_error", err.Error())
		return
	}
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		writeImageError(w, 400, "invalid_request_error", "invalid JSON")
		return
	}
	prompt := strings.TrimSpace(getStr(req, "prompt"))
	if prompt == "" {
		writeImageError(w, 400, "invalid_request_error", "prompt is required")
		return
	}

	// 每张图都是一次独立的上游调用，n 直接按倍数消耗账号额度，上限压到 4。
	n := 1
	if v, ok := req["n"].(float64); ok && int(v) > 0 {
		n = int(v)
	}
	if n > 4 {
		n = 4
	}

	_, modelCfg, err := resolveModel("gemini-image")
	if err != nil {
		writeImageError(w, 500, "api_error", err.Error())
		return
	}

	created := time.Now().Unix()
	wantURL := getStr(req, "response_format") == "url"
	data := make([]map[string]string, 0, n)
	for i := 0; i < n; i++ {
		text, _, res, err := callGemini(prompt, prompt, modelCfg, nil, nil, nil, nil)
		if err != nil {
			recordRequest("images.generations", "gemini-image", prompt, "", res, 502, err.Error(), false)
			writeImageError(w, 502, "upstream_error", "upstream error: "+err.Error())
			return
		}
		dataURL := dataURLRe.FindString(text)
		if dataURL == "" {
			recordRequest("images.generations", "gemini-image", prompt, text, res, 502, "no image in response", false)
			writeImageError(w, 502, "upstream_error", "no image in response")
			return
		}
		item := map[string]string{}
		if wantURL {
			item["url"] = dataURL
		} else {
			item["b64_json"] = dataURL[strings.Index(dataURL, ",")+1:]
		}
		data = append(data, item)
		recordRequest("images.generations", "gemini-image", prompt, text, res, 200, "", false)
	}
	writeJSON(w, 200, map[string]interface{}{"created": created, "data": data})
}

func writeImageError(w http.ResponseWriter, status int, typ, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{"message": msg, "type": typ},
	})
}
