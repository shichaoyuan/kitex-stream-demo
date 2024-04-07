namespace go chatbot

struct Request {
    1: optional string query,
}

struct Response {
    1: optional string event,
    2: optional string data,
}

service TestService {
    Response Chat (1: Request req) (streaming.mode="server"),
}
