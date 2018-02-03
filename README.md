HTTP Autoindex
==============

Lists all files and directories in the given path.

# Compile

```
docker build -t autoindex .
```

# Run

```
docker run --rm -it \
    -p 8080:8080 \
    -v `pwd`/files:/files \
    -e AUTOINDEX_ROOT=/files autoindex
```

## Sample nginx configuration

```
upstream autoindexer {
  server 192.168.1.80:8080;
}

server {
  listen 80;
  server_name example.tld;

  location / {
    proxy_pass http://autoindexer;
  }
}
```
