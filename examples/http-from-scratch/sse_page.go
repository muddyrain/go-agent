package main

import "net/http"

func eventsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	// 相对文件路径基于进程工作目录；本示例从项目根目录启动。
	http.ServeFile(w, r, "examples/http-from-scratch/sse_demo.html")
}
