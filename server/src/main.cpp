// This server serves both static and dynamic content.
// It opens two ports: plain HTTP on port 8000 and HTTP on port 8443.
// It implements the following endpoints:
//    /api/stats - respond with free-formatted stats on current connections
//    /api/f2/:id - wildcard example, respond with JSON string {"result": "URI"}
//    any other URI serves static files from s_root_dir
//

#include "mongoose.h"
#include "tiny-json.hxx"
#include <fstream>
#include <iostream>
#include <signal.h>

// Handle interrupts, like Ctrl-C
static int s_signo;
static void signal_handler(int signo) { s_signo = signo; }

static const char *s_debug_level = "2";
static const char *s_http_addr = "http://0.0.0.0:8000";   // HTTP port
static const char *s_https_addr = "https://0.0.0.0:8443"; // HTTPS port
static const char *s_root_dir = "./ui";
static const char *s_ssi_pattern = "#.html";

// We use the same event handler function for HTTP and HTTPS connections
// fn_data is nullptr for plain HTTP, and non-nullptr for HTTPS
static void fn(struct mg_connection *c, int ev, void *ev_data, void *fn_data)
{
  if (ev == MG_EV_ACCEPT && fn_data != nullptr) {
    mg_tls_opts opts{};
    opts.cert = "server.pem";    // Certificate PEM file
    opts.certkey = "server.pem"; // Certificate PEM file
    mg_tls_init(c, &opts);
  } else if (ev == MG_EV_HTTP_MSG) {
    mg_http_message *hm{(struct mg_http_message *)ev_data};
    if (mg_http_match_uri(hm, "/api/stats")) {
      // Print some statistics about currently established connections
      mg_printf(c, "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n");
      mg_http_printf_chunk(c, "ID PROTO TYPE      LOCAL           REMOTE\n");
      for (struct mg_connection *t = c->mgr->conns; t != nullptr; t = t->next) {
        char loc[40], rem[40];
        mg_http_printf_chunk(c, "%-3lu %4s %s %-15s %s\n", t->id, t->is_udp ? "UDP" : "TCP",
                             t->is_listening  ? "LISTENING"
                             : t->is_accepted ? "ACCEPTED "
                                              : "CONNECTED",
                             mg_straddr(&t->loc, loc, sizeof(loc)), mg_straddr(&t->rem, rem, sizeof(rem)));
      }
      mg_http_printf_chunk(c, ""); // Don't forget the last empty chunk
    } else if (mg_http_match_uri(hm, "/config")) {
      if (mg_vcasecmp(&hm->method, "POST") == 0) {
        std::ifstream jsonFile("config.json");
        std::string error{};
        auto json = TinyJson::JsonValue::parse(jsonFile, error);
        auto jsonStr = json.toString();
        mg_http_reply(c, 200, "Content-Type: application/json\r\n", jsonStr.c_str());
      } else {
        std::ifstream jsonFile("config.json");
        std::string error{};

        auto json = TinyJson::JsonValue::parse(jsonFile, error);
        if (!error.empty()) {
          char *errorStr = mg_mprintf("{%Q:%Q}", "error", error.c_str());
          mg_http_reply(c, 400, "Content-Type: application/json\r\n", errorStr);
          free(errorStr);
        } else {
          auto jsonStr = json.toString();
          mg_http_reply(c, 200, "Content-Type: application/json\r\n", jsonStr.c_str());
        }
      }
    } else {
      mg_http_message tmp{};
      mg_str unknown{mg_str_n("?", 1)}, *cl{};
      mg_http_serve_opts opts{};
      opts.root_dir = s_root_dir;
      opts.ssi_pattern = s_ssi_pattern;
      mg_http_serve_dir(c, hm, &opts);
      mg_http_parse((char *)c->send.buf, c->send.len, &tmp);
      cl = mg_http_get_header(&tmp, "Content-Length");
      if (cl == nullptr)
        cl = &unknown;
      MG_INFO(("%.*s %.*s %.*s %.*s", (int)hm->method.len, hm->method.ptr, (int)hm->uri.len, hm->uri.ptr, (int)tmp.uri.len, tmp.uri.ptr, (int)cl->len, cl->ptr));
    }
  }
  (void)fn_data;
}

static void usage(const char *prog)
{
  fprintf(stderr,
          "Mongoose v.%s\n"
          "Usage: %s OPTIONS\n"
          "  -d DIR    - directory to serve, default: '%s'\n"
          "  -h ADDR   - http address, default: '%s'\n"
          "  -s ADDR   - https address, default: '%s'\n"
          "  -v LEVEL  - debug level, from 0 to 4, default: '%s'\n",
          MG_VERSION, prog, s_root_dir, s_http_addr, s_https_addr, s_debug_level);
  exit(EXIT_FAILURE);
}

int main(int argc, char *argv[])
{
  char path[MG_PATH_MAX] = ".";
  struct mg_mgr mgr;
  int i;

  // Parse command-line flags
  for (i = 1; i < argc; i++) {
    if (strcmp(argv[i], "-d") == 0) {
      s_root_dir = argv[++i];
    } else if (strcmp(argv[i], "-h") == 0) {
      s_http_addr = argv[++i];
    } else if (strcmp(argv[i], "-s") == 0) {
      s_https_addr = argv[++i];
    } else if (strcmp(argv[i], "-v") == 0) {
      s_debug_level = argv[++i];
    } else {
      usage(argv[0]);
    }
  }

  // Root directory must not contain double dots. Make it absolute
  // Do the conversion only if the root dir spec does not contain overrides
  if (strchr(s_root_dir, ',') == nullptr) {
    realpath(s_root_dir, path);
    s_root_dir = path;
  }

  // Initialise stuff
  signal(SIGINT, signal_handler);
  signal(SIGTERM, signal_handler);
  mg_log_set(s_debug_level);
  mg_mgr_init(&mgr);
  if (mg_http_listen(&mgr, s_http_addr, fn, nullptr) == nullptr) {
    MG_ERROR(("Cannot listen on %s. Use http://ADDR:PORT or :PORT", s_http_addr));
    exit(EXIT_FAILURE);
  }
  if (mg_http_listen(&mgr, s_https_addr, fn, (void *)1) == nullptr) {
    MG_ERROR(("Cannot listen on %s. Use http://ADDR:PORT or :PORT", s_https_addr));
    exit(EXIT_FAILURE);
  }

  // Start infinite event loop
  MG_INFO(("Mongoose version : v%s", MG_VERSION));
  MG_INFO(("Listening on     : %s & %s", s_http_addr, s_https_addr));
  MG_INFO(("Web root         : [%s]", s_root_dir));
  while (s_signo == 0)
    mg_mgr_poll(&mgr, 1000);
  mg_mgr_free(&mgr);
  MG_INFO(("Exiting on signal %d", s_signo));
  return 0;
}
