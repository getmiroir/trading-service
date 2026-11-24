package clients

// ProxyHeader is the request header that carries the exchange proxy key
// (BinanceProxyKey, BybitProxyKey). Set at build time via -ldflags. When it or
// the key is empty, no proxy header is sent.
var ProxyHeader = ""

// setProxyHeader adds the proxy key to headers when both the header name and
// the key are configured.
func setProxyHeader(headers map[string]string, proxyKey *string) {
	if ProxyHeader == "" || proxyKey == nil {
		return
	}
	headers[ProxyHeader] = *proxyKey
}
