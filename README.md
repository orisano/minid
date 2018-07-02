# minid
minid is Dockerfile minifier.

## Features
* concatenate RUN command
* concatenate ENV command
* ...

## Installation
```bash
go get -u github.com/orisano/minid
```

## How to use
```bash
minid | docker build -f - .
```

## Author
Nao YONASHIRO (@orisano)

## License
MIT
