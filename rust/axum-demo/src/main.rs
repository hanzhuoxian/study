use axum::{
    routing::get,
    Router,
    response::Json,
};

use serde_json::{Value, json};

#[tokio::main]
async fn main() {
    // build our application with a single route
    let app = Router::new()
    .route("/", get(root))
    .route("/foo", get(get_foo).post(post_foo))
    .route("/foo/bar", get(foo_bar));

    // run our app with hyper, listening globally on port 3000
    let listener = tokio::net::TcpListener::bind("0.0.0.0:3000").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

// which calls one of these handlers
async fn root() -> Json<Value> {
    Json(json!({
        "message": "Hello, World."
    }))
}
async fn get_foo() {}
async fn post_foo() {}
async fn foo_bar() {}

