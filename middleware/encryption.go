package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"goals-api/pkg/crypto"

	"github.com/labstack/echo/v4"
)

type encPayload struct {
	Data string `json:"data"`
}

// bodyDumpWriter wraps http.ResponseWriter so we can capture whatever the
// handler writes, encrypt it, and flush a single encrypted JSON envelope.
type encryptedWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *encryptedWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *encryptedWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

// Encryption returns an Echo middleware that:
//  1. Reads the incoming body, expects {"data": "<base64 aes-gcm>"} when the
//     request is not empty, decrypts it and replaces the request body with
//     the plaintext JSON so downstream handlers work exactly as normal.
//  2. Captures whatever the handler writes as the response, encrypts it, and
//     sends back {"data": "<base64 aes-gcm>"} instead of the raw JSON.
//
// This gives end-to-end encrypted request/response bodies between the
// frontend and this API, on top of whatever transport security (HTTPS) is
// already in place.
func Encryption(box *crypto.Box) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			// ---- Decrypt request body (if present) ----
			if req.ContentLength > 0 || req.Body != nil {
				raw, err := io.ReadAll(req.Body)
				if err == nil && len(raw) > 0 {
					var env encPayload
					if err := json.Unmarshal(raw, &env); err == nil && env.Data != "" {
						plain, derr := box.Decrypt(env.Data)
						if derr != nil {
							return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid encrypted payload"})
						}
						req.Body = io.NopCloser(bytes.NewReader(plain))
						req.ContentLength = int64(len(plain))
					} else {
						// Not an encrypted envelope; restore original body as-is.
						req.Body = io.NopCloser(bytes.NewReader(raw))
					}
				}
			}

			// ---- Capture response, then encrypt it ----
			// We swap out the low-level io.Writer that echo.Response delegates
			// to, so the handler's normal c.JSON()/c.String() calls transparently
			// get buffered here instead of hitting the network.
			ew := &encryptedWriter{ResponseWriter: c.Response().Writer, status: http.StatusOK}
			originalWriter := c.Response().Writer
			c.Response().Writer = ew

			err := next(c)

			c.Response().Writer = originalWriter
			plainResp := ew.buf.Bytes()

			status := ew.status
			if status == 0 {
				status = c.Response().Status
			}
			if status == 0 {
				status = http.StatusOK
			}

			if len(plainResp) == 0 {
				return err
			}

			cipherText, encErr := box.Encrypt(plainResp)
			if encErr != nil {
				originalWriter.Header().Set(echo.HeaderContentType, "application/json; charset=UTF-8")
				originalWriter.WriteHeader(http.StatusInternalServerError)
				_, _ = originalWriter.Write([]byte(`{"error":"failed to encrypt response"}`))
				return nil
			}

			// IMPORTANT: write directly to the real ResponseWriter instead of
			// going through c.JSON()/c.Response() again. echo.Response.Committed
			// was already flipped to true by the handler's *first* c.JSON call
			// above (even though that call was silently captured into ew), so a
			// second c.JSON() here would be a no-op and the client would never
			// receive the encrypted envelope.
			body, marshalErr := json.Marshal(encPayload{Data: cipherText})
			if marshalErr != nil {
				originalWriter.Header().Set(echo.HeaderContentType, "application/json; charset=UTF-8")
				originalWriter.WriteHeader(http.StatusInternalServerError)
				_, _ = originalWriter.Write([]byte(`{"error":"failed to marshal response"}`))
				return nil
			}
			originalWriter.Header().Set(echo.HeaderContentType, "application/json; charset=UTF-8")
			originalWriter.WriteHeader(status)
			_, _ = originalWriter.Write(body)
			return nil
		}
	}
}
