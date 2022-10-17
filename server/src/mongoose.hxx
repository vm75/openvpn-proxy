#pragma once

#include "mongoose.h"
#include <map>
#include <string>

namespace Mongoose {

struct ServerOptions
{
  std::string debugDevel{"2"};
  std::string httpAddr{"http://0.0.0.0:8000"};   // HTTP port
  std::string httpsAddr{"https://0.0.0.0:8443"}; // HTTPS port
  std::string rootDir{"."};
  std::string ssiPattern{"#.html"};
  std::string certPath{};
  std::string certKeyPath{};
  std::string basicAuthUsername{};
  std::string basicAuthPasswd{};
};

enum class LogLevel : uint8_t
{
  NONE,
  ERROR,
  INFO,
  DEBUG,
  VERBOSE
};

enum class HttpMethod : uint8_t
{
  GET,
  PUT,
  POST,
  DELETE
};

enum class HttpReturnCode : uint16_t
{
  Continue = 100,
  Created = 201,
  Accepted = 202,
  NoContent = 204,
  PartialContent = 206,
  MovedPermanently = 301,
  Found = 302,
  NotModified = 304,
  BadRequest = 400,
  Unauthorized = 401,
  Forbidden = 403,
  NotFound = 404,
  ImATeapot = 418,
  InternalServerError = 500,
  NotImplemented = 501,
};

class HttpRequest
{
public:
  HttpRequest(void *eventData) noexcept : httpMessage(static_cast<mg_http_message *>(eventData)) {}

  bool isSecureEndpoint() const noexcept { return mg_url_is_ssl(httpMessage->uri.ptr); }

  HttpMethod getMethod() noexcept
  {
    static constexpr std::array<std::pair<std::string_view, HttpMethod>, 4> methods{{
        {"GET", HttpMethod::GET},
        {"PUT", HttpMethod::PUT},
        {"POST", HttpMethod::POST},
        {"DELETE", HttpMethod::DELETE},
    }};
    const std::string_view key{httpMessage->method.ptr};
    const auto itr = std::find_if(std::begin(methods), std::end(methods), [&key](const auto &v) { return v.first == key; });
    if (itr != std::end(methods)) {
      return itr->second;
    } else {
      return HttpMethod::GET; // defaults to GET
    }
  }

  bool uriMatches(const char *uri) noexcept { return mg_http_match_uri(httpMessage, uri); }

private:
  mg_http_message *httpMessage;
};

struct HttpResponse
{
  HttpReturnCode code{HttpReturnCode::Accepted};
  std::map<std::string, std::string> headers;
  std::string body;
  std::istream *data;

  // mg_http_reply
};

class Handler
{
public:
  virtual ~Handler() noexcept = default;

  virtual HttpResponse onHttpMessage(const HttpRequest &message) noexcept = 0;
};

template <bool b>
class Server_T
{
private:
  static void httpCallback(mg_connection *conn, int event, void *eventData, void *context) noexcept { dynamic_cast<Server_T<b>>(context)->callback(conn, event, eventData, false /*isSsl*/); }

  static void httpsCallback(mg_connection *conn, int event, void *eventData, void *context) noexcept { dynamic_cast<Server_T<b>>(context)->callback(conn, event, eventData, true /*isSsl*/); }

  // extra mime_types
  // ssl certificate

public:
  Server_T() noexcept
  {
    mg_log_set(options.debugDevel.c_str());
    mg_mgr_init(&manager);
    if (mg_http_listen(&manager, options.httpAddr.c_str(), httpCallback, this) == nullptr) {
      MG_ERROR(("Cannot listen on %s. Use http://ADDR:PORT or :PORT", options.httpAddr.c_str()));
    }
    if (mg_http_listen(&manager, options.httpsAddr.c_str(), httpsCallback, this) == nullptr) {
      MG_ERROR(("Cannot listen on %s. Use http://ADDR:PORT or :PORT", options.httpsAddr.c_str()));
    }
  }

  virtual ~Server_T() noexcept { mg_mgr_free(&manager); }

  void callback(mg_connection *conn, int eventType, void *eventData, bool isSsl) noexcept
  {
    switch (eventType) {
    case MG_EV_ACCEPT: {
      // use TLS if secure
      if (isSsl) {
        mg_tls_opts opts{};
        opts.cert = "server.pem";    // Certificate PEM file
        opts.certkey = "server.pem"; // Certificate PEM file
        mg_tls_init(conn, &opts);
      }
    } break;
    case MG_EV_HTTP_CHUNK: {
      // TODO
    } break;
    case MG_EV_HTTP_MSG: {
      HttpRequest req(eventData);
      if (req.uriMatches("/websocket")) {
        // Upgrade to websocket. From now on, a connection is a full-duplex
        // Websocket connection, which will receive MG_EV_WS_MSG events.
        mg_ws_upgrade(c, req.getRawMessage(), nullptr);
        break;
      }

      Handler *handler{};
      for (auto &entry : endpoints) {
        if (req.uriMatches(entry.first.c_str())) {
          continue;
        }
        handler = entry.second;
        break;
      }

      if (handler != nullptr) {
        auto resp = handler->onHttpMessage(req);
      } else {
        mg_http_serve_opts opts{};
        opts.root_dir = options.rootDir.c_str();
        opts.ssi_pattern = options.ssiPattern.c_str();
        // TODO extra mime types
        // TODO page 404
        mg_http_serve_dir(conn, hm, &opts);

#if DEBUG
        mg_http_message tmp{};
        mg_http_parse(static_cast<const char *>(conn->send.buf), conn->send.len, &tmp);

        mg_str *contentLen = mg_http_get_header(&tmp, "Content-Length");
        if (contentLen == nullptr) {
          static mg_str unknown{mg_str_n("?", 1)};
          contentLen = &unknown;
        }
        MG_INFO(("%.*s %.*s %.*s %.*s", (int)hm->method.len, hm->method.ptr, (int)hm->uri.len, hm->uri.ptr, (int)tmp.uri.len, tmp.uri.ptr, (int)cl->len, cl->ptr));
#endif
      }
    } break;
    case MG_EV_WS_MSG: {
      // TODO: Handle websocket
      // Got websocket frame. Received data is wm->data. Echo it back!
      struct mg_ws_message *wm = static_cast<mg_ws_message *>(eventData);
      mg_ws_send(c, wm->data.ptr, wm->data.len, WEBSOCKET_OP_TEXT);
      mg_iobuf_del(&c->recv, 0, c->recv.len);
    } break;
    case MG_EV_ERROR: {
      MG_ERROR(("Error in server: %s", static_cast<const char *>(eventData)));
    } break;
    }
  }

  void registerEndpoint(const std::string &endpoint, Handler *handler) noexcept
  {
    if (!endpoint.empty() && handler != nullptr) {
      endpoints[endpoint] = handler;
    }
  }

  int run() noexcept
  {
    isRunning = true;
    // Start infinite event loop
    MG_INFO(("Mongoose version : v%s", MG_VERSION));
    MG_INFO(("Listening on     : %s & %s", httpAddr.c_str(), httpsAddr.c_str()));
    MG_INFO(("Web root         : [%s]", s_root_dir));
    while (isRunning) {
      mg_mgr_poll(&manager, 1000);
    }
  }

  void stop() noexcept { isRunning = false; }

private:
  bool isRunning{};
  ServerOptions options;
  std::map<std::string, Handler *> endpoints;
  mg_mgr manager;
};

using Server = Server_T<true>;

} // namespace Mongoose