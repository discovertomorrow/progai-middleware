package usage

import "net/http"

type passthroughWriter struct {
	w      http.ResponseWriter
	update func(string)
	lb     []byte
	p      int
}

func (pw passthroughWriter) Write(p []byte) (int, error) {
	n := len(p)
	max := len(pw.lb) - 1
	for i := 0; i < n; i++ {
		if p[i] == '\n' || pw.p == max {
			line := string(pw.lb[0:pw.p])
			pw.p = 0
			pw.update(line)
		} else {
			pw.lb[pw.p] = p[i]
			pw.p = pw.p + 1
		}
	}
	return pw.w.Write(p)
}

func (pw passthroughWriter) Header() http.Header {
	return pw.w.Header()
}

func (pw passthroughWriter) Flush() {
	f, ok := pw.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (pw passthroughWriter) WriteHeader(statusCode int) {
	pw.w.WriteHeader(statusCode)
}
