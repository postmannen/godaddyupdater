FROM golang:alpine
RUN apk add git

RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/prometheus/client_golang/prometheus/promauto
RUN go get github.com/prometheus/client_golang/prometheus/promhttp

RUN mkdir /app 
ADD . /app/
WORKDIR /app 
RUN go build -o main .
RUN adduser -S -D -H -h /app appuser
USER appuser

# RUN go build -o main .
# CMD ["/app/main", "-proto=http://", "-host=eyer.io", "-port=80", "-hostListen=0.0.0.0"]
CMD ["/app/main", "-auth=env" ,"-checkInterval=99" ,"-domain=erter.org" ,"-subDomain=dev" ,"-promExpPort=9101"]
