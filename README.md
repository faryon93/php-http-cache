# About

A small background HTTP fetcher for PHP utilizing [goridge](https://github.com/spiral/goridge) as a high performance IPC bridge.

The request body is refreshed every `ttl` seconds.
When a cache entry is not access for the configured amount of time (command line flag `--timeout`), background fetching is disabled and the cache entry is removed.

## Usage
```php
$request = array(
    "method" => "POST",                 // method type (POST, GET allowed)
    "url" => "https://example.com",     // url                  
    "body" => http_build_query($body),  // body
    "ttl" => 12,                        // update interval in seconds
    "headers" => array(                 // request headers
        "Accept: application/json",
        "Content-Type: application/x-www-form-urlencoded",
        "User-Agent: Test"),
);

$rpc = new Goridge\RPC(new Goridge\SocketRelay("localhost", 6001));
$rpc->call("Cache.Request", json_encode($request))
```
