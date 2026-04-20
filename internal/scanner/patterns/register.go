package patterns

// Patterns are registered here in priority order — most specific first. The
// first pattern whose Matches returns true wins. Rationale per entry:
//
//   lighttpd:    unique "lighttpd/" footer string
//   caddy:       unique caddyserver.com footer link
//   tomcat:      unique "Directory Listing For" h1 / Apache-Coyote server
//   jetty:       unique "Directory: " h1 / Jetty server
//   iis:         unique "[To Parent Directory]" marker / Microsoft-IIS
//   python:      unique "Directory listing for" title / SimpleHTTP server
//   apache:      "Index of" h1 + <pre> or table#indexlist + Apache header
//   nginx:       "Index of" h1 + <pre> + nginx header
//   gofileserver: bare <pre> with <a> entries, no <title>/<h1>
//   generic:     loose fallback — any "Index of"/"Directory" heading, or
//                h1 == title (today's detector)
func init() {
	Register(Lighttpd{})
	Register(Caddy{})
	Register(Tomcat{})
	Register(Jetty{})
	Register(IIS{})
	Register(Python{})
	Register(Apache{})
	Register(Nginx{})
	Register(GoFileServer{})
	Register(Generic{})
}
