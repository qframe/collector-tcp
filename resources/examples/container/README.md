# TCP from a container


```bash
$ docker run -ti --name qframe-collector-tcp --rm -e SKIP_ENTRYPOINTS=1 \
             -v /var/run/docker.sock:/var/run/docker.sock qnib/uplain-golang bash
> execute CMD 'bash'
root@5326129ced40:/# curl -fsL  https://github.com/qnib/qframe-collector-tcp/releases/download/v0.1.1/qframe-collector-tcp > qframe-collector-tcp
root@5326129ced40:/# chmod +x qframe-collector-tcp
root@5326129ced40:/# ./qframe-collector-tcp
2017/04/24 21:48:09 [II] Dispatch broadcast for Back, Data and Tick
2017/04/24 21:48:09 [  INFO] docker-events >> Connected to 'moby' / v'17.04.0-ce'
2017/04/24 21:48:11 [  INFO] tcp >> Listening on 0.0.0.0:11001
#### Received 'Test-1493070504' from enriched container: {0xc4202df1e0 [{bind  /Users/kniepbert/src/github.com/qnib/qframe-collector-tcp/resources/examples/container /data   true }] 0xc420360000 0xc420106600}
root@3ece35256003:/usr/local/src/github.com/qnib/qframe-collector-tcp#
```

The message was sent like this...

```bash
$ docker run -ti -v $(pwd)/resources/examples/container/:/data/ \
         debian:latest /data/send-metric.sh $(docker inspect -f '{{ .NetworkSettings.Networks.bridge.IPAddress }}' qframe-collector-tcp)
```


